package service

import (
	"log/slog"
	"net"
	"net/netip"
	"reflect"
	"strconv"

	"github.com/SLP25/ESR/internal/packet"
)

type UDPServer struct {
	service *Service
	conn net.PacketConn
	closed bool
}

// Sends a packet to specified remote address
func (this *UDPServer) Send(p packet.Packet, address netip.AddrPort) error {
	slog.Info("Sending UDP message", "packet", reflect.TypeOf(p).Name(), "content", p, "addr", address.String())
	
	conn, err := net.Dial("udp", address.String())
	if err != nil { return err }

	_, err2 := conn.Write(packet.Serialize(p))
	err3 := conn.Close()

	if err2 != nil { return err2 }
	return err3
}


func (this *UDPServer) open(service *Service, port uint16) error {
	var err error
	*this = UDPServer{service: service}
	this.conn, err = net.ListenPacket("udp", ":" + strconv.FormatUint(uint64(port), 10))
	if err != nil {
		return err
	}

	slog.Info("Listening for UDP messages to", "port", port)
	go this.handle()
	return nil
}

func (this *UDPServer) close() error {
	this.closed = true
	return this.conn.Close()
}

func (this *UDPServer) handle() {
	for {
		buf := make([]byte, 1024)
		_, addr, err := this.conn.ReadFrom(buf)
		if err != nil {
			if this.closed {
				slog.Info("Closed UDP listener")
				return
			}

			slog.Error("Error receiving UDP message", err)
			continue
		}

		addrport, err := netip.ParseAddrPort(addr.String())
		if err != nil {
			slog.Error("Error parsing AddrPort from string", addr.String(), err)
			continue
		}

		packet := packet.Deserialize(buf)
		slog.Info("Received UDP message", "packet", reflect.TypeOf(packet).Name(), "content", packet, "addr", addr.String())
		this.service.sigQueue <- UDPMessage{packet: packet, addr: addrport, conn: this.conn}
	}
}