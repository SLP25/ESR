package service

import (
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"reflect"
	"strconv"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/utils"
)

type UDPPacket struct {
	Source netip.AddrPort
	Conn net.PacketConn
	Data []byte
}

type UDPServer struct {
	output chan UDPPacket
	conn net.PacketConn
	closed bool
}

// Sends a packet to specified remote address
func SendUDP(p packet.Packet, address netip.AddrPort) error {
	slog.Debug("Sending UDP message", "packet", reflect.TypeOf(p).Name(), "content", utils.Ellipsis(p, 50), "addr", address)
	
	conn, err := net.Dial("udp", address.String())
	if err != nil { return err }

	_, err2 := packet.Serialize(p, conn)
	err3 := conn.Close()

	if err2 != nil { return err2 }
	return err3
}


func (this *UDPServer) Open(port *uint16) error {
	if port == nil {
		return errors.New("UDPServer.open(): Nil port")
	}

	var err error
	*this = UDPServer{output: make(chan UDPPacket), closed: false}

	this.conn, err = net.ListenPacket("udp", ":" + strconv.FormatUint(uint64(*port), 10))
	if err != nil {
		return err
	}

	*port = netip.MustParseAddrPort(this.conn.LocalAddr().String()).Port()
	slog.Info("Listening for UDP messages", "port", *port)
	go this.handle()
	
	return nil
}

func (this *UDPServer) Close() error {
	if !this.closed {
		this.closed = true
		return this.conn.Close()
	} else {
		return nil
	}
}

func (this *UDPServer) Output() chan UDPPacket {
	return this.output
}

func (this *UDPServer) handle() {
	buf := make([]byte, 65600)
	
	for {
		n, addr, err := this.conn.ReadFrom(buf)
		
		if n != 0 {
			ans := make([]byte, n)
			copy(ans, buf)

			addrport := netip.MustParseAddrPort(addr.String())
			this.output <- UDPPacket{Source: addrport, Conn: this.conn, Data: ans}
		}

		if this.closed {
			slog.Info("Closed UDP listener")
			close(this.output)
			return
		} else if err != nil {
			slog.Error("Error receiving UDP message", "err", err)
			continue
		}
	}
}