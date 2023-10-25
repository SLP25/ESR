package service

import (
	"net"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/serialize"
)

type Init struct {}

type Closing struct {}



type Message interface {
	getPacket() packet.Packet
	getAddr() net.Addr
	sendResponse(packet.Packet) error
}


type UDPMessage struct {
	Packet packet.Packet
	Addr net.Addr
	udpServer net.PacketConn
}

func (this UDPMessage) getPacket() packet.Packet {
	return this.Packet
}

func (this UDPMessage) getAddr() net.Addr {
	return this.Addr
}

func (this UDPMessage) sendResponse(packet packet.Packet) error {
	_, err := this.udpServer.WriteTo(serialize.Serialize(packet), this.Addr)
	return err
}


type TCPMessage struct {
	Packet packet.Packet
	conn net.Conn
}

func (this TCPMessage) getPacket() packet.Packet {
	return this.Packet
}

func (this TCPMessage) getAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this TCPMessage) sendResponse(packet packet.Packet) error {
	_, err := this.conn.Write(serialize.Serialize(packet))
	return err
}

func (this TCPMessage) closeConn() error {
	return this.conn.Close()
}