package main

import (
	"os/exec"
	"os"
	"fmt"
	"net"
)

func main() {
	ffplay := exec.Command("ffplay", "-")
	stdin, _ := ffplay.StdinPipe()

	ffplay.Start()

	udpServer, err := net.ResolveUDPAddr("udp", ":1053")

	if err != nil {
		println("ResolveUDPAddr failed:", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialUDP("udp", nil, udpServer)
	if err != nil {
		println("Listen failed:", err.Error())
		os.Exit(1)
	}

	//close the connection
	defer conn.Close()

	_, err = conn.Write([]byte("This is a UDP message"))
	if err != nil {
		println("Write data failed:", err.Error())
		os.Exit(1)
	}

	// buffer to get data
	i := 0
	received := make([]byte, 188)
	for {
		_, err := conn.Read(received)
		fmt.Println(i)
		i++
		if err != nil {
			println("Read data failed:", err.Error())
			os.Exit(1)
		}
		stdin.Write(received)
	}
	fmt.Println("waiting")
	ffplay.Wait()
}