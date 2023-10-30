package main

import (
	"fmt"
	"net/netip"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)


type node struct {
    test int
}

var serv service.Service

func (this *node) processStartupResponse(p packet.StartupResponse) {
    if p.ConnectTo == netip.IPv4Unspecified() {
        fmt.Println("First node")
    } else {
        fmt.Println(p.ConnectTo.String())
    }
}

func (this *node) processTCPPacket(p packet.Packet) {
    fmt.Println("Packet received")
    switch p.(type) {
    case *packet.StartupResponse:
        fmt.Println("Startup response")
        this.processStartupResponse(*p.(*packet.StartupResponse))
    default:
        fmt.Println("Default")
        panic("Unsupported TCP packet")
    }
}

func (this *node) Handle(sig service.Signal) bool {
    fmt.Println("Aqui")
    switch sig.(type) {
    case service.Init:
        err := serv.TCPServer().SendConnect(packet.StartupRequest{Service: utils.Node}, netip.MustParseAddrPort("10.0.17.20:4002"))
        if err != nil {
            fmt.Println(err)
        }
    case service.TCPMessage:
        fmt.Println("Received packet")
        tcp := sig.(service.TCPMessage)
        packet := tcp.GetPacket()
        this.processTCPPacket(packet)
        //tcp.SendResponse(response)

    default:
        return false
    }

    return true
}

func main() {
    node := node{}
    serv.AddHandler(&node)
    
    errr := serv.Run(4002, 4002)
    fmt.Println(errr)
    fmt.Println("Hello! I'm a node")
}