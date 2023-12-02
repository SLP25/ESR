package main

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strconv"
	"sync"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)

var tcpPort uint16
var serv service.Service

type bootstrapper struct {
    config config
    accessNode netip.Addr
    mu sync.Mutex
}

func (this *bootstrapper) getConnectToIP() netip.AddrPort { //TODO: pick a good one instead of random
    this.mu.Lock()
    defer this.mu.Unlock()
    return utils.GetAnyValue(this.config.nodes, netip.AddrPortFrom(netip.IPv4Unspecified(), 0))
}

func (this *bootstrapper) Handle(sig service.Signal) bool {
    switch sig.(type) {
    case service.Init:
        return true

    case service.TCPMessage:
        msg := sig.(service.TCPMessage)
        switch msg.Packet().(type) {
        case packet.StartupRequest:
            req := msg.Packet().(packet.StartupRequest)
            switch req.Service {
            case utils.Client:
                utils.Warn(msg.SendResponse(packet.StartupResponseClient{ConnectTo: this.getConnectToIP()}))
                utils.Warn(msg.CloseConn())
                return true
            case utils.Node:
                resp, err := this.config.BootNode(msg.Addr().Addr())
                if err != nil {
                    slog.Error("Error starting node", "addr", msg.Addr(), "err", err)
                } else {
                    utils.Warn(msg.SendResponse(resp))
                    utils.Warn(msg.CloseConn())
                }
                return true
            }
        }        
    }
    
    return false
}

func main() {
    utils.SetupLogging()

    if len(os.Args) != 3 {
        fmt.Println("Usage: bootstrapper <port> <config>")
        return
    }

    aux, err := strconv.ParseUint(os.Args[1], 10, 16)
    if err != nil {
        fmt.Println("Invalid port: the port must be an integer between 0 and 65535")
        return
    }
    tcpPort = uint16(aux)

    bootstrapper := bootstrapper{config: MustReadConfig(os.Args[2])}

    serv.AddHandler(&bootstrapper)
    err = serv.Run(&tcpPort)
    if err != nil {
        slog.Error("Error running service", "err", err)
    }
}