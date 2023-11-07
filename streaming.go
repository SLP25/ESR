package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/AlexEidt/Vidio"
	"image"
	"image/jpeg"
	"os"
)

func main() {
	video, err := vidio.NewVideo("/home/rui-oliveira02/Videos/cpar.mp4")
	if err != nil {
		fmt.Println(err)
		return
	}

	img := image.NewRGBA(image.Rect(0, 0, video.Width(), video.Height()))
	video.SetFrameBuffer(img.Pix)

	file, _ := os.Create("output.bin")
	defer file.Close()
	// Error handling...
	frameCount := 0
	for video.Read() {
		// "frame" is a byte array storing the frame data in row-major order.
		// Each pixel is stored as 3 sequential bytes in RGB format.
		var b bytes.Buffer
		frameCompressed := bufio.NewWriter(&b)
		jpeg.Encode(frameCompressed, img, nil)

		size := make([]byte, 4)
		binary.LittleEndian.PutUint32(size, uint32(len(b.Bytes())))
		file.Write(size)
		file.Write(b.Bytes())
		frameCount++
	}

	fmt.Println(frameCount)
}
