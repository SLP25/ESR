package main

import (
	"fmt"
	//"github.com/asticode/go-astits"
	//"context"
	"net"
	"os/exec"
	//"time"
)

func main() {
	ffmpeg := exec.Command("ffmpeg", "-re", "-i", "video.mp4", "-c:v", "copy", "-preset", "ultrafast", "-c:a", "copy", "-f", "mpegts", "-")
	stdout, _ := ffmpeg.StdoutPipe()
	//f, _ := os.Open("video.ts")
	//defer f.Close()

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
	ffmpeg.Start()
	i := 0
	//frameDuration := 1000.0 / 1000.0
	for {
		packet := make([]byte, 1880)
		n, _ := stdout.Read(packet) //f.Read(packet)

		if n == 0 {
			//f.Seek(0, 0)
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
