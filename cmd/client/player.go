package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/utils"
)

type player struct {
	ffplay *exec.Cmd
	input chan<- packet.StreamPacket
	done <-chan struct{} //Used directly in client.go
}

func (this *player) PushPacket(p packet.StreamPacket) {
	if this.input != nil {
		this.input <- p
	}
}

func (this *player) Close() {
	if this.ffplay.Process != nil {
		this.ffplay.Process.Kill()
	}
}

func play(sdpConfig packet.StreamResponse) (*player, error) {
	ports := utils.FindFreePorts(2) //We pray that the next ports are also open
	sdpConfig.SetPorts(ports[0], ports[1])

	sdpTxt, err := sdpConfig.SDP.Marshal()
	if err != nil { return nil, err }

	videoConn, err := net.Dial("udp", "127.0.0.1:" + strconv.Itoa(int(ports[0])))
	if err != nil { return nil, err }
	videoCtrlConn, err := net.Dial("udp", "127.0.0.1:" + strconv.Itoa(int(ports[0] + 1)))
	if err != nil { return nil, err }
	audioConn, err := net.Dial("udp", "127.0.0.1:" + strconv.Itoa(int(ports[1])))
	if err != nil { return nil, err }
	audioCtrlConn, err := net.Dial("udp", "127.0.0.1:" + strconv.Itoa(int(ports[1] + 1)))
	if err != nil { return nil, err }
	
	
	done := make(chan struct{}, 1)
	input := make(chan packet.StreamPacket, 100)
	player := player{
		ffplay: exec.Command("ffplay", "-window_title", streamID, "-protocol_whitelist", "pipe,udp,rtp", "-f", "sdp", "-i", "-"),
		done: done,
		input: input,
	}
	
	stdin, _ := player.ffplay.StdinPipe()
	
	stderr, _ := player.ffplay.StderrPipe()
	go func() {
		for {
			aux := make([]byte, 500)
			n, _ := stderr.Read(aux)
			fmt.Fprint(os.Stderr, string(aux[:n]))
		}
	}()
	

	err = player.ffplay.Start()
	if err != nil { return nil, err }
	
	_, err = stdin.Write(sdpTxt)
	if err != nil {
		player.Close()
		return nil, err
	}

	err = stdin.Close()
	if err != nil {
		player.Close()
		return nil, err
	}

	go func() {
		utils.Warn(player.ffplay.Wait())
		player.input = nil
		close(input)
		done <- struct{}{}
		close(done)
	}()

	go func() {
		for p := range input {
			var err error

			switch p.Type {
				case packet.Video:
					_, err = videoConn.Write(p.Content)
				case packet.VideoControl:
					_, err = videoCtrlConn.Write(p.Content)
				case packet.Audio:
					_, err = audioConn.Write(p.Content)
				case packet.AudioControl:
					_, err = audioCtrlConn.Write(p.Content)
			}

			if err != nil {}
			//utils.Warn(err)

			//TODO: We can't always log the error, since we need to wait for ffplay to initialize
			//Before that, attempting to send a packet will result in connection refused
		}
	}()

	return &player, nil
}