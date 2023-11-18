package main

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strconv"
	"time"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)


var port uint16
var bootAddr netip.AddrPort
var serv service.Service


type server struct {
    streams config
    cancelStreams map[string] chan any
}



func (this *server) Handle(sig service.Signal) bool {

    switch sig.(type) {
    case service.TCPMessage:
        msg := sig.(service.TCPMessage)

        switch msg.Packet().(type) {
        case packet.ProbeRequest:
            p := msg.Packet().(packet.ProbeRequest)

            if s, ok := this.streams[p.StreamID]; ok {
                slog.Debug("ProbeRequest", "file", s)
                metadata := utils.StreamMetadata{Throughput: 10} //TODO: get throughput of stream
                //TODO: if the connection doesn't have the bandwidth?

                utils.Warn(msg.SendResponse(p.RespondExistant(metadata)))
            } else {
                utils.Warn(msg.SendResponse(p.RespondNonExistant()))
            }
            return true

        case packet.StreamRequest:
            p := msg.Packet().(packet.StreamRequest)
            s, ok := this.streams[p.StreamID]
            if ok {
                slog.Debug("StreamRequest", "file", s)
                this.cancelStreams[p.StreamID] = make(chan any)
                go func() {
                    defer delete(this.cancelStreams, p.StreamID)

                    end := time.After(300 * time.Second)

                    //TODO
                    for {
                        serv.UDPServer().Send(packet.StreamPacket{StreamID: p.StreamID, Content:[]byte{1,2,3}}, netip.AddrPortFrom(msg.Addr().Addr(), p.Port))

                        select {
                        case <-this.cancelStreams[p.StreamID]:
                            return
                        case <-end:
                            msg.SendResponse(packet.StreamEnd{StreamID: p.StreamID})
                            return
                        case <-time.After(1 * time.Second):
                        }
                    }
                }()
            } else {
                utils.Warn(msg.SendResponse(packet.StreamEnd{}))
                utils.Warn(msg.CloseConn())
            }
            return true

        case packet.StreamCancel:
            p := msg.Packet().(packet.StreamCancel)
            if c, ok := this.cancelStreams[p.StreamID]; ok {
                c <- true
            }
            return true
        }    
    }

	return false
}

func main() {
    utils.SetupLogging()

    if len(os.Args) != 4 {
        fmt.Println("Usage: server <port> <bootAddr> <config>")
        return
    }

    aux, err := strconv.ParseUint(os.Args[1], 10, 16)
    if err != nil {
        fmt.Println("Invalid port: the port must be an integer between 0 and 65535")
        return
    }
    port = uint16(aux)

    bootAddr, err = netip.ParseAddrPort(os.Args[2])
    if err != nil {
        fmt.Println("Invalid boot address:", err)
        return
    }

    server := server{streams: MustReadConfig(os.Args[3]), cancelStreams: make(map[string]chan any)}

    serv.AddHandler(&server)
    err = serv.Run(&port, nil)
    if err != nil {
        slog.Error("Error running service", "err", err)
    }
}