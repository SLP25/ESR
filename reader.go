package main

import (
	"encoding/binary"
	"fmt"
	"github.com/AlexEidt/Vidio"
	"image"
	"os"
	"os/exec"
	"io"
	"math/rand"
	"syscall"
)

func main() {
	video, err := vidio.NewVideo("/home/rui-oliveira02/Videos/cpar.mp4")
	if err != nil {
		fmt.Println(err)
		return
	}

	img := image.NewRGBA(image.Rect(0, 0, video.Width(), video.Height()))
	video.SetFrameBuffer(img.Pix)

	file, _ := os.Open("output.bin")
	defer file.Close()

	cmd := exec.Command("ffplay", "fifo")
	syscall.Mkfifo("fifo", 0640)

	// Start the ffmpeg command
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting ffplay:", err)
		return
	}

	videoIn, err := os.OpenFile("fifo", os.O_WRONLY, 0640)

	if err != nil {
		fmt.Println("Error creating pipe for stdin:", err)
		return
	}

	// Error handling...
	frameCount := 0
	for {
		sizeBuffer := make([]byte, 4)
		_, err := file.Read(sizeBuffer)

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Println(err)
			return
		}

		size := binary.LittleEndian.Uint32(sizeBuffer)
		data := make([]byte, size)
		file.Read(data)
		random := rand.Intn(100)

		if random > 0 && frameCount != 0 {
			videoIn.Write(data)
		}
		
		frameCount++
	}

	videoIn.Close()
	fmt.Println(frameCount)
}
