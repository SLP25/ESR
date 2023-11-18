package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/netip"
	"os"
	"time"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)

var port uint16
var bootAddr netip.AddrPort
var streamID string
var serv service.Service

type client struct {
    accessNode netip.AddrPort
}

func (this client) Handle(sig service.Signal) bool {
    switch sig.(type) {
    case service.Init:
        request := packet.StartupRequest{Service: utils.Client}
        response, err := service.InterceptTCPResponse[packet.StartupResponseClient](&serv, request, bootAddr)
        if err != nil {
            slog.Error("Error on Init", "err", err)
            serv.Close()
            return true
        }
        utils.Warn(serv.TCPServer().CloseConn(bootAddr.Addr()))
        this.accessNode = response.ConnectTo

        randInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
        req := packet.StreamRequest{StreamID: streamID, RequestID: randInt, Port: port}
        serv.TCPServer().SendConnect(req, this.accessNode)

        select {
        case <- service.InterceptTCPPackets[packet.StreamEnd](&serv, this.accessNode, 1):
            fmt.Println("Stream ended/doesn't exist")
            serv.Close()

        case <- service.InterceptSignal[service.Closing](&serv, 1): //TODO: weird bug once inside CastChan (received nil)
        }

        return true

    case service.UDPMessage:
        msg := sig.(service.UDPMessage)
        if msg.Addr().Addr() == this.accessNode.Addr() {
            switch msg.Packet().(type) {
            case packet.StreamPacket:
                //TODO
                fmt.Println("Stream packet received:", msg.Packet().(packet.StreamPacket).Content)
                return true
            }
        }
    }
    return false
}

func main() {
    utils.SetupLogging()

    if len(os.Args) != 3 {
        fmt.Println("Usage: client <bootAddr> <streamID>")
        return
    }

    var err error
    bootAddr, err = netip.ParseAddrPort(os.Args[1])
    if err != nil {
        fmt.Println("Invalid boot address:", err)
        return
    }

    streamID = os.Args[2]

    client := client{}
    serv.AddHandler(client)

    err = serv.Run(nil, &port)
    if err != nil {
        slog.Error("Error running service", "err", err)
    }
}