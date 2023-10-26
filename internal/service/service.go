package service

import (
	"bufio"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"sync"

	"github.com/SLP25/ESR/internal/packet"
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

func (this *Service) SendUDP(p packet.Packet, address netip.AddrPort) {
	this.udpServer.WriteTo(packet.Serialize(p), addr{network: "udp", addrport: address})
}

func (this *Service) Run(tcpPort int, udpPort int) error { //make ports optional (in case only one listener is needed)

	this.finished.Add(1)
	var err error

	this.tcpServer, err = net.Listen("tcp", ":" + strconv.Itoa(tcpPort))
	if err != nil {
		return err
	}
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

		packet := packet.Deserialize(netData)
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

		addrport, err := netip.ParseAddrPort(addr.String())
		if err != nil {
			fmt.Println(err) //TODO: error handling
			return
		}

		packet := packet.Deserialize(buf) //TODO: confirmar se isto nao está a passar bytes vazios

		this.Handler.Handle(UDPMessage{Packet: packet, Addr: addrport, udpServer: this.udpServer})
	}
}