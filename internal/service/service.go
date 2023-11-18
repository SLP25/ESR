package service

import (
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
	handling sync.WaitGroup
}

func (this *Service) UDPServer() *UDPServer {
	return &this.udpServer
}

func (this *Service) TCPServer() *TCPServer {
	return &this.tcpServer
}

func (this *Service) Run(tcpPort *uint16, udpPort *uint16) error {
	var err error

	this.sigQueue = make(chan Signal, 20)
	err = this.tcpServer.open(this, tcpPort)
	if err != nil { return err }
	defer this.tcpServer.close()

	err = this.udpServer.open(this, udpPort)
	if err != nil { return err }
	defer this.udpServer.close()
	
	go this.handle(Init{})

	for sig := range this.sigQueue {
		this.paused.Wait()
		go this.handle(sig)
	}

	this.handle(Closing{})
	this.handling.Wait()
	return nil
}

func (this *Service) Close() {
	close(this.sigQueue)
}

func (this *Service) Enqueue(s Signal) {
	this.sigQueue <- s
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

	slog.Warn("RemoveHandler found no such handler to remove", "handler", h)
	return false
}

//blocks future calls to service.handle() until the function returns
func (this *Service) PauseHandleWhile(f func()) {
	this.paused.Add(1)
	defer this.paused.Add(-1)
	f()
}

func PauseHandleWhile[T any](this *Service, f func() T) T {
	this.paused.Add(1)
	defer this.paused.Add(-1)
	return f()
}

func (this *Service) handle(sig Signal) {
	this.handling.Add(1)
	defer this.handling.Add(-1)

	for i := len(this.handlers)-1; i >= 0; i-- { //TODO: loop sus (concorrencia)
		if (this.handlers[i].Handle(sig)) {
			return
		}
	}

	slog.Warn("Unprocessed signal", "type", reflect.TypeOf(sig).Name(), "content", sig)
}