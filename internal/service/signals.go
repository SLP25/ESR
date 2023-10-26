package service

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/SLP25/ESR/internal/packet"
)

type Init struct {}

type Closing struct {}



type Message interface {
	getPacket() packet.Packet
	getAddr() netip.AddrPort
	sendResponse(packet.Packet) error
}


type UDPMessage struct {
	Packet packet.Packet
	Addr netip.AddrPort
	udpServer net.PacketConn
}

func (this UDPMessage) getPacket() packet.Packet {
	return this.Packet
}

func (this UDPMessage) getAddr() netip.AddrPort {
	return this.Addr
}

func (this UDPMessage) sendResponse(p packet.Packet) error {
	_, err := this.udpServer.WriteTo(packet.Serialize(p), addr{network: "udp", addrport: this.Addr})
	return err
}


type TCPMessage struct {
	Packet packet.Packet
	conn net.Conn
}

func (this TCPMessage) getPacket() packet.Packet {
	return this.Packet
}

func (this TCPMessage) getAddr() netip.AddrPort {
	addr, err := netip.ParseAddrPort(this.conn.RemoteAddr().String())
	if (err != nil) {
		fmt.Println(err) //TODO: error handling
	}

	return addr
}

func (this TCPMessage) sendResponse(p packet.Packet) error {
	_, err := this.conn.Write(packet.Serialize(p))
	return err
}

func (this TCPMessage) closeConn() error {
	return this.conn.Close()
}