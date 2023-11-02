package main

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)


type node struct {
    parent netip.AddrPort
    runningStreams map[string] []netip.AddrPort
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
    switch p.(type) {
    default:
        slog.Warn("Unsupported TCP packet", p)
    }
}

func (this *node) Handle(sig service.Signal) bool {
    switch sig.(type) {
    case service.Init:
        addr := netip.MustParseAddrPort("127.0.0.1:4002")
        err := serv.TCPServer().SendConnect(packet.StartupRequest{Service: utils.Node}, addr)
        if err != nil {
            slog.Error("Error on Init:", err)
            return true
        }
        resp := service.InterceptTCPPackets[packet.StartupResponse](&serv, addr, 1)
        go func() {
            this.processStartupResponse(<-resp)
        }()

    case service.TCPMessage:
        tcp := sig.(service.TCPMessage)
        packet := tcp.Packet()
        this.processTCPPacket(packet)
        //tcp.SendResponse(response)

    default:
        return false
    }

    return true
}

func main() {
    fmt.Println("Hello! I'm a node")

    handler := slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
    log := slog.New(slog.NewTextHandler(os.Stdout, &handler))
    slog.SetDefault(log)

    node := node{}
    serv.AddHandler(&node)
    
    err := serv.Run(4003, 4003)
    if err != nil {
        slog.Error("Error running service:", err)
    }
}