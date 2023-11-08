package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"syscall"
)

func sendAudio(audioFIFO *os.File) {
	file, _ := os.Open("audio.mp3")

	for {
		buffer := make([]byte, 4096)
		n, err := file.Read(buffer)
		if n == 0 || err == io.EOF {
			return
		}

		if err != nil {
			fmt.Println(err)
		}

		audioFIFO.Write(buffer)
	}
}

func createFIFOS() (string, string) {
	pid := os.Getpid()
	audioFIFO := fmt.Sprintf("/tmp/esr_audio%d", pid)
	videoFIFO := fmt.Sprintf("/tmp/esr_video%d", pid)

	syscall.Mkfifo(audioFIFO, 0640)
	syscall.Mkfifo(videoFIFO, 0640)

	return audioFIFO, videoFIFO
}

func openFIFOS(audioFIFO string, videoFIFO string) (*os.File, *os.File, error) {
	fmt.Println("Start")
	audioFile, err := os.Create(audioFIFO)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("End")
	videoFile, err := os.Create(videoFIFO)
	if err != nil {
		return nil, nil, err
	}

	return audioFile, videoFile, nil
}

func closeFIFOS(audioFile *os.File, videoFile *os.File) {
	audioFile.Close()
	videoFile.Close()
}

func cleanFIFOS(audioFIFO string, videoFIFO string) {
	syscall.Unlink(audioFIFO)
	syscall.Unlink(videoFIFO)
}

func startProcesses(audioFIFO string, videoFIFO string) (*exec.Cmd, *exec.Cmd, error) {
	ffmpeg := exec.Command("ffmpeg", "-i", videoFIFO, "-i", audioFIFO, "-c:v", "h264", "-preset", "ultrafast", "-c:a", "aac", "-f", "matroska", "-")
	ffplay := exec.Command("ffplay", "-")

	ffplay_stdin, _ := ffplay.StdinPipe()
	ffmpeg.Stdout = ffplay_stdin

	if err := ffmpeg.Start(); err != nil {
		return ffmpeg, ffplay, err
	}
	if err := ffplay.Start(); err != nil {
		return ffmpeg, ffplay, err
	}

	return ffmpeg, ffplay, nil
}

func waitForProcesses(ffmpeg *exec.Cmd, ffplay *exec.Cmd) {
	ffmpeg.Wait()
	ffplay.Wait()
}

func main() {
	file, _ := os.Open("output.bin")
	defer file.Close()

	audioFIFO, videoFIFO := createFIFOS()
	defer cleanFIFOS(audioFIFO, videoFIFO)

	ffmpeg, ffplay, err := startProcesses(audioFIFO, videoFIFO)
	if err != nil {
		fmt.Println("Unable to start child processes")
		fmt.Println(err)
		return
	}

	audioFile, videoFile, err := openFIFOS(audioFIFO, videoFIFO)
	defer closeFIFOS(audioFile, videoFile)
	if err != nil {
		fmt.Println("Unable to open FIFOs")
		fmt.Println(err)
		return
	}

	go sendAudio(audioFile)

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
			videoFile.Write(data)
		}

		frameCount++
	}

	waitForProcesses(ffmpeg, ffplay)
}
