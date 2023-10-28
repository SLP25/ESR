package service

import (
	"net"
	"net/netip"

	"github.com/SLP25/ESR/internal/packet"
)

type Signal interface {
	//type
	//embedded subclass? change to interface??
}

type Init struct {}

type Closing struct {}

type Message interface {
	Packet() packet.Packet
	Addr() netip.AddrPort
	SendResponse(packet.Packet) error
	CloseConn() error //Not necessary for udp
}


type TCPMessage struct {
	Packet packet.Packet
	conn net.Conn
}

func (this TCPMessage) GetPacket() packet.Packet {
	return this.Packet
}

func (this TCPMessage) GetAddr() netip.AddrPort {
	addr, err := netip.ParseAddrPort(this.conn.RemoteAddr().String())
	if (err != nil) {
		panic(err)
	}

	return addr
}

func (this TCPMessage) SendResponse(p packet.Packet) error {
	_, err := this.conn.Write(packet.Serialize(p))
	return err
}

func (this TCPMessage) CloseConn() error {
	return this.conn.Close()
}


type UDPMessage struct {
	packet packet.Packet
	addr netip.AddrPort
	conn net.PacketConn
}

func (this UDPMessage) GetPacket() packet.Packet {
	return this.packet
}

func (this UDPMessage) GetAddr() netip.AddrPort {
	return this.addr
}

func (this UDPMessage) SendResponse(p packet.Packet) error {
	_, err := this.conn.WriteTo(packet.Serialize(p), addr{network: "udp", addrport: this.addr})
	return err
}

func (this UDPMessage) CloseConn() error {
	return this.conn.Close()
}