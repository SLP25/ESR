package main

import (
	"fmt"
	//"github.com/asticode/go-astits"
	//"context"
	"os"
	"net"
	"os/exec"
	//"time"
)

func main() {
	//ffmpeg := exec.Command("ffmpeg", "-i", "video.mp4", "-c:v", "copy", "-preset", "ultrafast", "-c:a", "copy", "-f", "mpegts", "video.ts")
	//ffmpeg.Start()
	//ffmpeg.Wait()

	f, _ := os.Open("video.ts")
	defer f.Close()

	udpServer, err := net.ListenPacket("udp", ":1053")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer udpServer.Close()

	ffplay := exec.Command("ffplay", "-af", "'volume=0.0'", "-")
	stdin, _ := ffplay.StdinPipe()

	ffplay.Start()

	buf := make([]byte, 1024)
	_, addr, err := udpServer.ReadFrom(buf)
	if err != nil {
		return
	}

	i := 0
	//frameDuration := 1000.0 / 1000.0
	for {
		packet := make([]byte, 188)
		n, _ := f.Read(packet)

		if n == 0 {
			f.Seek(0,0)
			continue
		}
		stdin.Write(packet)
		udpServer.WriteTo(packet, addr)
		i++
		//time.Sleep(time.Duration(frameDuration) * time.Millisecond)
	}
	fmt.Println(i)
	//stdin.Close()
	//ffplay.Wait()
}
