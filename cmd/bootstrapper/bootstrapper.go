package main

import (
    "fmt"
    "github.com/SLP25/ESR/internal/service"
    "github.com/SLP25/ESR/internal/utils"
    "github.com/SLP25/ESR/internal/packet"
    "net/netip"
    "sync"
)

var serv service.Service

type bootstrapper struct {
    accessNode netip.Addr
    nodes []netip.Addr
    mu sync.Mutex
}

func (this *bootstrapper) getConnectToIP() netip.Addr {
    ip := netip.IPv4Unspecified()

    this.mu.Lock()
    if len(this.nodes) > 0 {
        ip = this.nodes[0]
    } 
    this.mu.Unlock()

    return ip
}

func (this *bootstrapper) processStartupRequest(p packet.StartupRequest) packet.StartupResponse {
    switch p.Service {
    case utils.Node:
        return packet.StartupResponse{ConnectTo: this.getConnectToIP()}
    default:
        panic("Not supported")
    }
}

func (this *bootstrapper) processTCPPacket(p packet.Packet) packet.Packet {
    fmt.Println("Packet received")
    switch p.(type) {
    case packet.StartupRequest:
        fmt.Println("Startup request")
        return this.processStartupRequest(p.(packet.StartupRequest))
    default:
        fmt.Println("Default")
        panic("Unsupported TCP packet")
    }
}

func (this *bootstrapper) Handle(sig service.Signal) {
    switch sig.(type) {
    case service.Init:
        fmt.Println("Ready!")
        //TODO: connect to bootstrapper
        //this.accessNode = ...

    case service.TCPMessage:
        fmt.Println("Received packet")
        tcp := sig.(service.TCPMessage)
        packet := tcp.GetPacket()
        response := this.processTCPPacket(packet)
        tcp.SendResponse(response)
    }
}

func main() {
    bootstrapper := bootstrapper{}
    serv = service.Service{Handler: &bootstrapper}
    errr := serv.Run(4002, 4002)
    fmt.Println(errr)
    fmt.Println("Hello! I'm the bootstrapper")
}