package service

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/serialize"
)


type Signal interface {
	//type
	//embedded subclass? change to interface??
}


type Handler interface {
	Handle(Signal)
}

type Service struct {
	Handler Handler
	udpServer net.PacketConn
	tcpServer net.Listener
	finished sync.WaitGroup
}

func (this *Service) SendUDP(packet packet.Packet, addr net.Addr) {
	this.udpServer.WriteTo(serialize.Serialize(packet), addr)
}

func (this *Service) Run(tcpPort int, udpPort int) error { //make ports optional (in case only one listener is needed)

	this.finished.Add(1)

	s1, err := net.Listen("tcp", ":" + strconv.Itoa(tcpPort))
	if err != nil {
		return err
	}
	this.tcpServer = s1
	defer this.tcpServer.Close()
	go this.handleTCPServer()
	
	s2, err := net.ListenPacket("udp", ":" + strconv.Itoa(udpPort))
	if err != nil {
		return err
	}
	this.udpServer = s2
	defer this.udpServer.Close()
	go this.handleUDPServer()
	
	this.Handler.Handle(Init{})
	this.finished.Wait()
	this.Handler.Handle(Closing{})
	return nil
}

func (this *Service) Close() {
	this.finished.Done()
}

func (this *Service) handleTCPServer() {
	for {
		c, err := this.tcpServer.Accept()
		if err != nil {
			fmt.Println(err) //TODO: error handling
			return
		}
		go this.handleConnection(c)
	}
}

func (this *Service) handleConnection(c net.Conn) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(c).ReadBytes(0)
		if err != nil {
				fmt.Println(err) //TODO: error handling
				return
		}

		packet := serialize.Deserialize(netData)
		this.Handler.Handle(TCPMessage{Packet: packet, conn: c})
	}
}

func (this *Service) handleUDPServer() {
	for {
		buf := make([]byte, 1024)
		_, addr, err := this.udpServer.ReadFrom(buf)
		if err != nil {
			fmt.Println(err) //TODO: error handling
			return
		}

		packet := serialize.Deserialize(buf) //TODO: confirmar se isto nao está a passar bytes vazios
		this.Handler.Handle(UDPMessage{Packet: packet, Addr: addr, udpServer: this.udpServer})
	}
}