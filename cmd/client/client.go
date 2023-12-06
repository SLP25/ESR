package main

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"time"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)

var udpPort uint16
var bootAddr netip.AddrPort
var streamID string
var serv service.Service

type client struct {
    accessNode netip.AddrPort
    player *player
}

func (this *client) Handle(sig service.Signal) bool {
    switch sig.(type) {
    case service.Init:
        request := packet.StartupRequest{Service: utils.Client}
        fmt.Println("Searching for access node...")
        response, err := service.InterceptTCPResponse[packet.StartupResponseClient](&serv, request, bootAddr) //TODO: timeout
        utils.Warn(serv.TCPServer().CloseConn(bootAddr.Addr()))
        
        if err != nil {
            slog.Error("Error on Init", "err", err)
            fmt.Println("Couldn't connect to bootstrapper. Terminating")
            serv.Close()
            return true
        }
        
        if !response.ConnectTo.IsValid() {
            fmt.Println("Invalid response. This could indicate the network has no active nodes")
            serv.Close()
            return true
        }

        var streamEnd chan packet.StreamEnd
        var servClosing <-chan service.Closing

        serv.PauseHandleWhile(func() {
            this.accessNode = response.ConnectTo
            fmt.Println("Access node address received:", this.accessNode)
            fmt.Println("Connecting to access node...")
    
            randInt := utils.RandID()
            req := packet.StreamRequest{StreamID: streamID, RequestID: randInt, Port: udpPort}
            serv.TCPServer().SendConnect(req, this.accessNode)
            
            fmt.Println("Waiting for node response...")
            
            servClosing = service.InterceptSignal[service.Closing](&serv, 1)
            streamEnd = utils.MapChan[service.Signal, packet.StreamEnd](service.Intercept(&serv, func(sig service.Signal) bool {
                msg, ok := sig.(service.TCPMessage)
                if !ok { return false }

                p, ok := msg.Packet().(packet.StreamEnd)
                return ok && p.StreamID == streamID
            }, 1), func(sig service.Signal) packet.StreamEnd {
                return sig.(service.TCPMessage).Packet().(packet.StreamEnd)
            })
        })

        select {
            case <- streamEnd:
                fmt.Println("Stream '" + streamID + "' doesn't exist")
                serv.Close()
                return true

            case msg := <-service.InterceptTCPPackets[packet.StreamResponse](&serv, this.accessNode, 1):
                fmt.Println("Response received! Loading video player...")
                var err error
                this.player, err = play(msg)
    
                if err != nil {
                    slog.Error("Failed to start player", "err", err)
                    serv.Close()
                    return true
                }

                go func() {
                    <-this.player.done
                    fmt.Println("Video player terminated")
                    serv.Close()
                }()

            case <- servClosing:
                return true
        }

        select {
            case <- streamEnd:
                fmt.Println("Stream ended")
                time.Sleep(time.Millisecond * 200)
                serv.Close()
            case <- servClosing:
        }

        this.player.Close()
        return true

    case service.TCPDisconnected:
        disc := sig.(service.TCPDisconnected)

        if !this.accessNode.IsValid() || disc.Addr().Addr() != this.accessNode.Addr() { return false }

        fmt.Println("Access node disconnected. Terminating")
        serv.Close()
        return true

    case service.UDPMessage:
        msg := sig.(service.UDPMessage)

        if msg.Addr().Addr() != this.accessNode.Addr() { return false }

        p, ok := msg.Packet().(packet.StreamPacket)
        if !ok { return false }

        if this.player != nil {
            this.player.PushPacket(p)
        }
        
        return true
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
    serv.AddHandler(&client)

    err = serv.Run(nil, &udpPort)
    if err != nil {
        slog.Error("Error running service", "err", err)
    }
}