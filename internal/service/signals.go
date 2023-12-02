package service

import (
	"bytes"
	"log/slog"
	"net"
	"net/netip"
	"reflect"

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

type TCPConnected struct {
	conn *connection
}

func (this TCPConnected) Addr() netip.AddrPort {
	return netip.MustParseAddrPort(this.conn.RemoteAddr().String())
}

func (this TCPConnected) Send(p packet.Packet) error {
	slog.Info("Sending TCP message", "packet", reflect.TypeOf(p).Name(), "content", p, "addr", this.conn.RemoteAddr())
	_, err := packet.Serialize(p, this.conn)
	return err
}

func (this TCPConnected) CloseConn() error {
	this.conn.closed = true
	return this.conn.Close()
}


type TCPDisconnected struct {
	remoteAddr netip.AddrPort
}

func (this TCPDisconnected) Addr() netip.AddrPort {
	return this.remoteAddr
}


type TCPMessage struct {
	packet packet.Packet
	conn *connection
}

func (this TCPMessage) Packet() packet.Packet {
	return this.packet
}

func (this TCPMessage) Addr() netip.AddrPort {
	return netip.MustParseAddrPort(this.conn.RemoteAddr().String())
}

func (this TCPMessage) SendResponse(p packet.Packet) error {
	_, err := packet.Serialize(p, this.conn)
	slog.Debug("Sending TCP message", "packet", reflect.TypeOf(p).Name(), "content", p, "addr", this.conn.RemoteAddr())
	return err
}

func (this TCPMessage) CloseConn() error {
	this.conn.closed = true
	return this.conn.Close()
}


type UDPMessage struct {
	packet packet.Packet
	localPort uint16
	addr netip.AddrPort
	conn net.PacketConn
}

func (this UDPMessage) Packet() packet.Packet {
	return this.packet
}

func (this UDPMessage) LocalPort() uint16 {
	return netip.MustParseAddrPort(this.conn.LocalAddr().String()).Port()
}

func (this UDPMessage) Addr() netip.AddrPort {
	return this.addr
}

func (this UDPMessage) SendResponse(p packet.Packet) error {
	data := make([]byte, 35600)
	buf := bytes.NewBuffer(data)
	n, err := packet.Serialize(p, buf)

	_, err = this.conn.WriteTo(data[:n], addr{network: "udp", addrport: this.addr})
	slog.Debug("Sending UDP message", "packet", reflect.TypeOf(p).Name(), "content", p, "addr", this.addr)
	return err
}

func (this UDPMessage) CloseConn() error {
	return this.conn.Close()
}