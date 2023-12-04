package service

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/utils"
)

type connection struct {
	net.Conn
	closed bool
}

type TCPServer struct {
	output chan Signal
	listener net.Listener
	conns map[netip.Addr]*connection
	connsMutex sync.RWMutex
	closed bool
}

// Establishes a TCP connection to the specified remote address.
// If such connection is already established, nothing happens (this method is idempotent)
func (this *TCPServer) Connect(addr netip.AddrPort) error {
	this.connsMutex.Lock()
	defer this.connsMutex.Unlock()

	_, ok := this.conns[addr.Addr()]
	if ok { return nil }

	slog.Info("Connecting to remote", "addr", addr)
	conn, err := net.Dial("tcp", addr.String())
	if err != nil { return err }

	c := &connection{conn, false}
	this.conns[addr.Addr()] = c
	go this.handleConnection(c)
	return nil
}

// Sends a packet to the specified remote address.
// If the connection wasn't established beforehand, the operation fails
func (this *TCPServer) Send(p packet.Packet, addr netip.Addr) error {
	slog.Debug("Sending TCP message", "packet", reflect.TypeOf(p).Name(), "content", utils.Ellipsis(p, 50), "addr", addr)

	this.connsMutex.RLock()
	defer this.connsMutex.RUnlock()

	_, ok := this.conns[addr]
	if !ok { return errors.New("Error sending TCP packet: connection not established to remote " + addr.String()) }

	_, err := packet.Serialize(p, this.conns[addr])
	return err
}

// Closes the connection to specified remote address.
// If no such connection exists, nothing happens (this method is idempotent)
func (this *TCPServer) CloseConn(addr netip.Addr) error {
	this.connsMutex.Lock()
	defer this.connsMutex.Unlock()
	
	conn, ok := this.conns[addr]
	if !ok { return nil }

	conn.closed = true
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


func (this *TCPServer) Open(port *uint16) error {
	var err error
	*this = TCPServer{output: make(chan Signal), conns: make(map[netip.Addr]*connection)}

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

func (this *TCPServer) Close() error {
	if !this.closed {
		this.closed = true
		close(this.output)
	}
	
	if this.listener != nil {
		return this.listener.Close()
	} else {
		return nil
	}
}

func (this *TCPServer) Output() chan Signal {
	return this.output
}

func (this *TCPServer) sendOutput(msg Signal) {
	if !this.closed {
		this.output <- msg
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

		this.connsMutex.Lock()
		c := &connection{conn, false}
		this.conns[addr.Addr()] = c
		this.connsMutex.Unlock()

		go this.handleConnection(c)
	}
}

func (this *TCPServer) handleConnection(c *connection) {
	remote := netip.MustParseAddrPort(c.RemoteAddr().String())
	slog.Info("Listening for TCP messages from", "addr", c.RemoteAddr())

	this.output <- TCPConnected{c}
	defer func() {
		slog.Info("Stopped listening for TCP messages from", "addr", c.RemoteAddr())
		c.closed = true
		this.connsMutex.Lock()
		delete(this.conns, remote.Addr())
		this.connsMutex.Unlock()
		this.sendOutput(TCPDisconnected{remote})
	}()

	for {
		packet, err := packet.Deserialize(c)
		time.Sleep(time.Millisecond * 10)

		if errors.Is(err, io.EOF) { //closed by remote
			slog.Info("TCP connection closed by remote", "addr", c.RemoteAddr())
			return
		} else if c.closed { //closed by local
			return
		} else if err != nil {
			slog.Error("Error receiving TCP message from", "addr", c.RemoteAddr(), "err", err)
			utils.Warn(c.Close())
			return
		}

		slog.Debug("Received TCP message", "addr", c.RemoteAddr(), "packet", reflect.TypeOf(packet).Name(), "content", utils.Ellipsis(packet, 50))
		this.sendOutput(TCPMessage{packet: packet, conn: c})
	}
}