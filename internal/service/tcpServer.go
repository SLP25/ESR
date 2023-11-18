package service

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"reflect"
	"strconv"

	"github.com/SLP25/ESR/internal/packet"
)

type TCPServer struct {
	service *Service
	listener net.Listener
	conns map[netip.Addr]net.Conn
	closed bool
}

// Establishes a TCP connection to the specified remote address.
// If such connection is already established, nothing happens (this method is idempotent)
func (this *TCPServer) Connect(addr netip.AddrPort) error {
	_, ok := this.conns[addr.Addr()]
	if ok { return nil }

	slog.Info("Connecting to remote", "addr", addr)
	conn, err := net.Dial("tcp", addr.String())
	if err != nil { return err }

	this.conns[addr.Addr()] = conn
	go this.handleConnection(conn)
	return nil
}

// Sends a packet to the specified remote address.
// If the connection wasn't established beforehand, the operation fails
func (this *TCPServer) Send(p packet.Packet, addr netip.Addr) error {
	slog.Info("Sending TCP message", "packet", reflect.TypeOf(p).Name(), "content", p, "addr", addr)

	_, ok := this.conns[addr]
	if !ok { return errors.New("Error sending packet: connection not established to remote " + addr.String()) }

	_, err := this.conns[addr].Write(packet.Serialize(p))
	return err
}

// Closes the connection to specified remote address.
// If no such connection exists, nothing happens (this method is idempotent)
func (this *TCPServer) CloseConn(addr netip.Addr) error {
	conn, ok := this.conns[addr]
	if !ok { return nil }

	err := conn.Close()
	if err != nil { return err }

	delete(this.conns, addr)
	return nil
}

// Sends a packet to the specified remote address.
// If no connection to remote address was active, one is established and left dangling
func (this *TCPServer) SendConnect(p packet.Packet, addr netip.AddrPort) error {
	err := this.Connect(addr)
	if err != nil { return err }

	return this.Send(p, addr.Addr())
}

// Sends a packet to the specified remote address and closes the connection.
// If the connection wasn't established beforehand, the operation fails.
// Whether the operation is successful or not, the connection is closed
func (this *TCPServer) SendLast(p packet.Packet, addr netip.Addr) error {
	err := this.Send(p, addr)
	err2 := this.CloseConn(addr)

	if err != nil { return err }
	return err2
}

// Sends a packet to the specified remote address and closes the connection.
// If no connection to remote address was active, one is established automatically
// Whether the operation is successful or not, the connection is closed
func (this *TCPServer) SendSingle(p packet.Packet, addr netip.AddrPort) error {
	err := this.Connect(addr)
	if err != nil { return err }

	err2 := this.Send(p, addr.Addr())
	err3 := this.CloseConn(addr.Addr())

	if err2 != nil { return err2 }
	return err3
}


func (this *TCPServer) open(service *Service, port *uint16) error {
	var err error
	*this = TCPServer{service: service, conns: make(map[netip.Addr]net.Conn)}

	if port != nil {
		this.listener, err = net.Listen("tcp", ":" + strconv.FormatUint(uint64(*port), 10))
		if err != nil {
			return err
		}
		
		*port = netip.MustParseAddrPort(this.listener.Addr().String()).Port()
		slog.Info("Listening for TCP connections", "port", *port)
		go this.handle()
	}

	return nil
}

func (this *TCPServer) close() error {
	this.closed = true
	if this.listener != nil {
		return this.listener.Close()
	} else {
		return nil
	}
}

func (this *TCPServer) handle() {
	for {
		conn, err := this.listener.Accept()
		if err != nil {

			if this.closed {
				slog.Info("Closed TCP listener")
				return
			}

			slog.Error("Error accepting connection", "err", err)
			continue
		}

		addr, err2 := netip.ParseAddrPort(conn.RemoteAddr().String())
		if err2 != nil {
			slog.Error("Error parsing AddrPort from string", "addr", conn.RemoteAddr(), "err", err)
			continue
		}

		this.conns[addr.Addr()] = conn
		go this.handleConnection(conn)
	}
}

func (this *TCPServer) handleConnection(c net.Conn) {
	slog.Info("Listening for TCP messages from", "addr", c.RemoteAddr())

	this.service.Enqueue(TCPConnected{c})
	defer this.service.Enqueue(TCPDisconnected{netip.MustParseAddrPort(c.RemoteAddr().String())})
	defer delete(this.conns, netip.MustParseAddrPort(c.RemoteAddr().String()).Addr())

	for {
		netData, err := bufio.NewReader(c).ReadBytes(0)
		if errors.Is(err, io.EOF) {
			slog.Info("TCP connection closed by remote", "addr", c.RemoteAddr())
			return
		} else if err != nil { //TODO: not log an error if connection is closed by local
			slog.Error("Error receiving TCP message from", "addr", c.RemoteAddr(), "err", err)
			return
		}
		packet := packet.Deserialize(netData)
		slog.Info("Received TCP message", "addr", c.RemoteAddr(), "packet", reflect.TypeOf(packet).Name(), "content", packet)
		this.service.Enqueue(TCPMessage{packet: packet, conn: c})
	}
}