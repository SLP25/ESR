package service

import (
	"fmt"
	"sync"
)


type Handler interface {
	Handle(Signal) bool
}

type Service struct {
	handlers []Handler
	udpServer UDPServer
	tcpServer TCPServer
	finished sync.WaitGroup
}

func (this *Service) UDPServer() UDPServer {
	return this.udpServer
}

func (this *Service) TCPServer() TCPServer {
	return this.tcpServer
}

func (this *Service) Run(tcpPort uint16, udpPort uint16) error { //make ports optional (in case only one listener is needed)
	this.finished.Add(1)
	var err error

	this.tcpServer, err = openTCP(this, tcpPort)
	if err != nil { return err }
	defer this.tcpServer.close()

	this.udpServer, err = openUDP(this, udpPort)
	if err != nil { return err }
	defer this.udpServer.close()
	
	this.handle(Init{})
	this.finished.Wait()
	this.handle(Closing{})
	return nil
}

func (this *Service) Close() {
	this.finished.Done()
}

func (this *Service) AddHandler(h Handler) {
	this.handlers = append(this.handlers, h)
}

// Removes the topmost instance of the specified handler from the handler stack
func (this *Service) RemoveHandler(h Handler) bool {

	for i := len(this.handlers)-1; i >= 0; i-- {
		if this.handlers[i] == h {
			this.handlers = append(this.handlers[:i], this.handlers[i+1:]...)
			return true
		}
	}

	return false
}

func (this *Service) handle(sig Signal) {
	for i := len(this.handlers)-1; i >= 0; i-- {
		if (this.handlers[i].Handle(sig)) {
			return
		}
	}

	fmt.Println("Warning: unprocessed signal ", sig)
}