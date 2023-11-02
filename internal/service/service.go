package service

import (
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)


type Handler interface {
	Handle(Signal) bool
}

type Service struct {
	handlers []Handler	//TODO: controlo de concorrencia
	udpServer UDPServer
	tcpServer TCPServer
	sigQueue chan Signal
	paused sync.WaitGroup
}

func (this *Service) UDPServer() *UDPServer {
	return &this.udpServer
}

func (this *Service) TCPServer() *TCPServer {
	return &this.tcpServer
}

func (this *Service) Run(tcpPort uint16, udpPort uint16) error { //make ports optional (in case only one listener is needed)
	var err error

	this.sigQueue = make(chan Signal, 20)
	err = this.tcpServer.open(this, tcpPort)
	if err != nil { return err }
	defer this.tcpServer.close()

	err = this.udpServer.open(this, udpPort)
	if err != nil { return err }
	defer this.udpServer.close()
	
	this.handle(Init{})
	
	for sig := range this.sigQueue {
		this.paused.Wait()
		this.handle(sig)
	}

	this.handle(Closing{})
	return nil
}

func (this *Service) Close() {
	close(this.sigQueue)
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

	slog.Warn("RemoveHandler found no such handler to remove", h)
	return false
}

func (this *Service) PauseHandleWhile(f func()) {
	this.paused.Add(1)
	f()
	this.paused.Add(-1)
}

func (this *Service) handle(sig Signal) {
	fmt.Println(len(this.handlers))

	for i := len(this.handlers)-1; i >= 0; i-- {
		fmt.Println(i)
		if (this.handlers[i].Handle(sig)) {
			fmt.Println("Processed.")
			return
		}
	}

	slog.Warn("Unprocessed signal", "type", reflect.TypeOf(sig).Name(), "content", sig)
}