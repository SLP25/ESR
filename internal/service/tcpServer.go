package service

import (
	"bufio"
	"fmt"
	"net"
	"net/netip"
	"strconv"

	"github.com/SLP25/ESR/internal/packet"
)

type TCPServer struct {
	service *Service
	listener net.Listener
	conns map[netip.AddrPort]net.Conn
}

// Establishes a TCP connection to the specified remote address.
// If such connection is already established, nothing happens (this method is idempotent)
func (this TCPServer) Connect(addr netip.AddrPort) error {
	_, ok := this.conns[addr]
	if ok { return nil }

	conn, err := net.Dial("tcp", addr.String())
	if err != nil { return err }
	this.conns[addr] = conn
	return nil
}

// Sends a packet to the specified remote address.
// If the connection wasn't established beforehand, the operation fails
func (this TCPServer) Send(p packet.Packet, addr netip.AddrPort) error {
	_, err := this.conns[addr].Write(packet.Serialize(p))
	return err
}

// Closes the connection to specified remote address.
// If no such connection exists, nothing happens (this method is idempotent)
func (this TCPServer) CloseConn(addr netip.AddrPort) error {
	conn, ok := this.conns[addr]
	if !ok { return nil }

	err := conn.Close()
	if err != nil { return err }

	delete(this.conns, addr)
	return nil
}

// Sends a packet to the specified remote address.
// If no connection to remote address was active, one is established and left dangling
func (this TCPServer) SendConnect(p packet.Packet, addr netip.AddrPort) error {
	err := this.Connect(addr)
	if err != nil { return err }

	return this.Send(p, addr)
}

// Sends a packet to the specified remote address and closes the connection.
// If the connection wasn't established beforehand, the operation fails.
// Whether the operation is successful or not, the connection is closed
func (this TCPServer) SendLast(p packet.Packet, addr netip.AddrPort) error {
	err := this.Send(p, addr)
	err2 := this.CloseConn(addr)

	if err != nil { return err }
	return err2
}

// Sends a packet to the specified remote address and closes the connection.
// If no connection to remote address was active, one is established automatically
// Whether the operation is successful or not, the connection is closed
func (this TCPServer) SendSingle(p packet.Packet, addr netip.AddrPort) error {
	err := this.Connect(addr)
	if err != nil { return err }

	err2 := this.Send(p, addr)
	err3 := this.CloseConn(addr)

	if err2 != nil { return err2 }
	return err3
}


func openTCP(service *Service, port uint16) (TCPServer, error) {
	var err error
	this := TCPServer{service: service, conns: make(map[netip.AddrPort]net.Conn)}

	this.listener, err = net.Listen("tcp", ":" + strconv.FormatUint(uint64(port), 10))
	if err != nil {
		return this, err
	}
	
	go this.handle()
	return this, nil
}

func (this TCPServer) close() error {
	return this.listener.Close()
}

func (this TCPServer) handle() {
	for {
		conn, err := this.listener.Accept() //TODO: return if listener is closed
		if err != nil {
			fmt.Println(err)
			continue
		}

		addr, err2 := netip.ParseAddrPort(conn.RemoteAddr().String())
		if err2 != nil {
			fmt.Println(err2)
			continue
		}

		this.conns[addr] = conn
		go this.handleConnection(conn)
	}
}

func (this TCPServer) handleConnection(c net.Conn) {
	fmt.Printf("Serving %s\n", c.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(c).ReadBytes(0)
		if err != nil {
			fmt.Println(err)
			return
		}

		packet := packet.Deserialize(netData)
		go this.service.handle(TCPMessage{Packet: packet, conn: c})
	}
}