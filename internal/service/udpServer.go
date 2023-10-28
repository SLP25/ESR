package service

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"

	"github.com/SLP25/ESR/internal/packet"
)

type UDPServer struct {
	service *Service
	conn net.PacketConn
}

// Sends a packet to specified remote address
func (this UDPServer) Send(p packet.Packet, address netip.AddrPort) error {
	conn, err := net.Dial("udp", address.String())
	if err != nil { return err }

	_, err2 := conn.Write(packet.Serialize(p))
	err3 := conn.Close()

	if err2 != nil { return err2 }
	return err3
}


func openUDP(service *Service, port uint16) (UDPServer, error) {
	var err error
	this := UDPServer{service: service}
	this.conn, err = net.ListenPacket("udp", ":" + strconv.FormatUint(uint64(port), 10))
	if err != nil {
		return this, err
	}

	go this.handle()
	return this, nil
}

func (this UDPServer) close() error {
	return this.conn.Close()
}

func (this UDPServer) handle() {
	for {
		buf := make([]byte, 1024)
		_, addr, err := this.conn.ReadFrom(buf) //TODO: return if listener is closed
		if err != nil {
			fmt.Println(err)
			continue
		}

		addrport, err := netip.ParseAddrPort(addr.String())
		if err != nil {
			fmt.Println(err)
			continue
		}

		packet := packet.Deserialize(buf) //TODO: confirmar se isto nao está a passar bytes vazios
		go this.service.handle(UDPMessage{packet: packet, addr: addrport, conn: this.conn})
	}
}