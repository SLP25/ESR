package main

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"reflect"
	"sync"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
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
    fmt.Println(reflect.TypeOf(p))
    switch p.(type) {
    case packet.StartupRequest:
        fmt.Println("Startup request")
        return this.processStartupRequest(p.(packet.StartupRequest))
    default:
        fmt.Println("Default")
        panic("Unsupported TCP packet")
    }
}

func (this *bootstrapper) Handle(sig service.Signal) bool {
    switch sig.(type) {
    case service.Init:
        fmt.Println("Ready!")

    case service.TCPMessage:
        fmt.Println("Received packet")
        tcp := sig.(service.TCPMessage)
        packet := tcp.Packet()
        response := this.processTCPPacket(packet)
        tcp.SendResponse(response)
    default:
        return false
    }
    
    return true
}

func main() {
    fmt.Println("Hello! I'm the bootstrapper")

    handler := slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
    log := slog.New(slog.NewTextHandler(os.Stdout, &handler))
    slog.SetDefault(log)

    bootstrapper := bootstrapper{}
    serv.AddHandler(&bootstrapper)
    err := serv.Run(4002, 4002)
    if err != nil {
        slog.Error("Error running service", err)
    }
}