package main

import (
	"bufio"
	"fmt"
	"io"
	"net/netip"
	"os/exec"
	"strconv"
	"time"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
	"github.com/pion/sdp/v2"
	"github.com/vansante/go-ffprobe"
)

//Represents a continuous stream of video, started at a specific time, that loops forever.
//Clients can subscribe to receive stream packets.
//If no clients are subscribed, the background ffmpeg process is stopped to save resources.
type stream struct {
	streamID string
	filepath string
	loop bool

	startTime time.Time
	metadata utils.StreamMetadata
	duration time.Duration

	client netip.AddrPort

	cancelChan chan struct{} 	//if null, ffmpeg is down
	canceledChan chan struct{}
}

//Returns the moment in the video file the stream is currently transmitting
func (this *stream) currentTime() time.Duration {
	return time.Now().Sub(this.startTime) % this.duration
}

func start(streamID string, filepath string, loop bool) (*stream, error) {
	stream := &stream{
		streamID: streamID,
		filepath: filepath,
		loop: loop,
		startTime: time.Now(),
	}

	data, err := ffprobe.GetProbeData(filepath, 5 * time.Second)
	if err != nil { return nil, err }

	for _, s := range data.Streams {
		d, err := strconv.ParseFloat(s.Duration, 64)
		if err != nil { return nil, err }

		b, err := strconv.Atoi(s.BitRate)
		if err != nil { return nil, err }

		stream.duration = max(stream.duration, time.Duration(d * 1000000000))
		stream.metadata.Bitrate += b
	}

	return stream, nil
}

func (this *stream) setClient(client netip.AddrPort) (sdp.SessionDescription, error) {
	this.terminate()
	this.client = client
	return this.startBackground()
}

func (this *stream) removeClient() {
	this.terminate()
	this.client = netip.AddrPort{}
}

func (this *stream) moveCurrentTime(current time.Duration) error {
	if this.cancelChan != nil {
		this.terminate()
		this.startTime = time.Now().Add(-current)
		_, err := this.startBackground() //ignore sdp since it isn't changed
		return err
	} else {
		this.startTime = time.Now().Add(-current)
		return nil
	}
}

//stops the background ffmpeg process
func (this *stream) terminate() {
	if this.cancelChan != nil {
		this.cancelChan <- struct{}{}
		<- this.canceledChan
	}
}

func formatDuration(d time.Duration) string {
	hours := d.Truncate(time.Hour) / time.Hour
	minutes := (d.Truncate(time.Minute) - d.Truncate(time.Hour)) / time.Minute
	seconds := (d.Truncate(time.Second) - d.Truncate(time.Minute)) / time.Second
	milliseconds := (d.Truncate(time.Millisecond) - d.Truncate(time.Second)) / time.Millisecond

	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
}

func readSDP(r io.Reader) (sdp.SessionDescription, error) {
	scanner := bufio.NewScanner(r)

	//ignore first line
	if !scanner.Scan() { 
		err := scanner.Err()
		if err == nil { err = io.EOF }
		return sdp.SessionDescription{}, io.EOF
	}

	txt := ""

	for {
		if !scanner.Scan() {
			err := scanner.Err()
			if err == nil { err = io.EOF }
			return sdp.SessionDescription{}, io.EOF
		}
		
		line := scanner.Text()
		if line == "" { break } //break on empty line
		txt += line + "\n"
	}

	session := sdp.SessionDescription{}
	err := session.Unmarshal([]byte(txt))
	return session, err
}

func (this *stream) startBackground() (sdp.SessionDescription, error) {
	
	sdpChan := make(chan sdp.SessionDescription)
	errorChan := make(chan error)

	go func() {

		var vidPort, audPort uint16
		var vidServer, audServer, vidCtrlServer, audCtrlServer service.UDPServer

		err := vidServer.Open(&vidPort)
		if err != nil { errorChan <- err; return }
		defer vidServer.Close()

		err = audServer.Open(&audPort)
		if err != nil { errorChan <- err; return }
		defer audServer.Close()

		vidCtrlPort := vidPort + 1
		err = vidCtrlServer.Open(&vidCtrlPort)
		if err != nil { errorChan <- err; return }
		defer vidCtrlServer.Close()

		audCtrlPort := audPort + 1
		err = audCtrlServer.Open(&audCtrlPort)
		if err != nil { errorChan <- err; return }
		defer audCtrlServer.Close()


		offset := formatDuration(this.currentTime())
		args := []string{"-re", "-ss", offset}

		if this.loop {
			args = append(args, []string{"-stream_loop", "-1"}...)
		}

		args = append(args, "-i", this.filepath)
		args = append(args, "-vcodec", "copy", "-an", "-f", "rtp", "rtp://127.0.0.1:" + strconv.FormatUint(uint64(vidPort), 10))
		args = append(args, "-acodec", "copy", "-vn", "-f", "rtp", "rtp://127.0.0.1:" + strconv.FormatUint(uint64(audPort), 10))

		ffmpeg := exec.Command("ffmpeg", args...)
		stdout, _ := ffmpeg.StdoutPipe()
		ffmpeg.Start()
		this.cancelChan = make(chan struct{}, 1)
		this.canceledChan = make(chan struct{}, 1)

		defer func() {
			if ffmpeg.Process != nil {
				ffmpeg.Process.Kill()
			}
			aux := this.canceledChan

			this.cancelChan = nil
			this.canceledChan = nil

			aux <- struct{}{}			
		}()

		//read sdp from stdout
		sdp, err := readSDP(stdout)
		if err != nil { errorChan <- err; return }

		sdpChan <- sdp

		//TODO: wait for natural end to ffmpeg (ffmpeg.Wait(); etc...)

		for {
			var p packet.StreamPacket
			select {
				case msg := <- vidServer.Output():
					p = packet.StreamPacket{Type: packet.Video, Content: msg.Data}
				case msg := <- audServer.Output():
					p = packet.StreamPacket{Type: packet.Audio, Content: msg.Data}
				case msg := <- vidCtrlServer.Output():
					p = packet.StreamPacket{Type: packet.VideoControl, Content: msg.Data}
				case msg := <- audCtrlServer.Output():
					p = packet.StreamPacket{Type: packet.AudioControl, Content: msg.Data}
				case <-this.cancelChan:
					return
			}
	
			utils.Warn(service.SendUDP(p, this.client))
		}
	}()

	select {
		case sdp := <-sdpChan:
			return sdp, nil
		case err := <-errorChan:
			return sdp.SessionDescription{}, err
	}
}
