package service

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"reflect"
	"sync"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/utils"
)


type Handler interface {
	Handle(Signal) bool
}

type handlerNode struct {
	Handler
	removed bool
}

type Service struct {
	handlers []*handlerNode
	handlersMutex sync.Mutex
	udpServers map[uint16]*UDPServer
	tcpServer TCPServer
	sigQueue chan Signal
	paused sync.WaitGroup
	handling sync.WaitGroup
	closed bool
	closing sync.Mutex
}

func (this *Service) TCPServer() *TCPServer {
	return &this.tcpServer
}

func (this *Service) UDPServer(port uint16) *UDPServer {
	return this.udpServers[port]
}

func (this *Service) AddUDPServer(port *uint16) error {
	if port == nil {
		return errors.New("Service.AddUDPServer(): Nil UDP port")
	} else if _, ok := this.udpServers[*port]; ok {
		return errors.New(fmt.Sprint("Service.AddUDPServer(): Duplicated UDP ports:", *port))
	}

	var server UDPServer
	err := server.Open(port)
	if err != nil { return err }
	go func() {
		for msg := range server.Output() {
			if this.closed { return }
			
			packet, err := packet.Deserialize(bytes.NewReader(msg.Data))
			if err != nil {
				slog.Error("Error receiving UDP message from", "addr", msg.Source, "err", err)
				continue
			}

			localPort := netip.MustParseAddrPort(msg.Conn.LocalAddr().String()).Port()
			//slog.Debug("Received UDP message", "addr", msg.Source, "packet", reflect.TypeOf(packet).Name(), "content", utils.Ellipsis(packet, 50))
			this.sigQueue <- UDPMessage{packet: packet, localPort: localPort, addr: msg.Source, conn: msg.Conn}
		}
	}()
	this.udpServers[*port] = &server
	return nil
}

func (this *Service) RemoveUDPServer(port uint16) error {
	if server, ok := this.udpServers[port]; ok {
		return server.Close()
	} else {
		return errors.New(fmt.Sprint("service.RemoveUDPServer(): called on closed port", port))
	}
}

//returns when all Handles return
func (this *Service) Run(tcpPort *uint16, udpPorts... *uint16) error {
	var err error

	this.sigQueue = make(chan Signal, 20)
	this.udpServers = make(map[uint16]*UDPServer)
	err = this.tcpServer.Open(tcpPort)
	if err != nil { return err }
	go func() {
		for msg := range this.tcpServer.Output() {
			if this.closed { return }
			this.sigQueue <- msg
		}
	}()

	defer func() {
		utils.Warn(this.tcpServer.Close())

		for _, server := range this.udpServers {
			utils.Warn(server.Close())
		}
	}()

	for _, port := range udpPorts {
		err := this.AddUDPServer(port)
		if err != nil { return err }
	}
	
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
	this.closing.Lock()
	defer this.closing.Unlock()

	if !this.closed {
		this.closed = true
		close(this.sigQueue)
	}
}

func (this *Service) Enqueue(s Signal) bool {
	this.closing.Lock()
	defer this.closing.Unlock()
	
	if !this.closed {
		this.sigQueue <- s
	}

	return !this.closed
}

func (this *Service) AddHandler(h Handler) {
	this.handlers = append(this.handlers, &handlerNode{h, false})
}

// Removes the topmost instance of the specified handler from the handler stack
func (this *Service) RemoveHandler(h Handler) bool {

	for i := len(this.handlers)-1; i >= 0; i-- {
		if this.handlers[i].Handler == h {
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

	this.handlersMutex.Lock()
	handlers := make([]*handlerNode, len(this.handlers))
	copy(handlers, this.handlers)
	this.handlersMutex.Unlock()

	for i := len(handlers)-1; i >= 0; i-- {
		if (!handlers[i].removed && handlers[i].Handle(sig)) {
			return
		}
	}

	slog.Warn("Unprocessed signal", "type", reflect.TypeOf(sig).Name(), "content", sig)
}