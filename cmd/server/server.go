package main

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strconv"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)


var port uint16 //both tcp (for control msgs) and udp (for pings)
var bootAddr netip.AddrPort
var serv service.Service


type server struct {
    streams map[string]*stream
}


func (this *server) Handle(sig service.Signal) bool {

    switch sig.(type) {
    case service.TCPMessage:
        msg := sig.(service.TCPMessage)

        switch msg.Packet().(type) {
        case packet.ProbeRequest:
            p := msg.Packet().(packet.ProbeRequest)

            if s, ok := this.streams[p.StreamID]; ok {
                utils.Warn(msg.SendResponse(p.RespondExistant(s.metadata)))
            } else {
                utils.Warn(msg.SendResponse(p.RespondNonExistant()))
            }
            return true

        case packet.StreamRequest:
            p := msg.Packet().(packet.StreamRequest)
            s, ok := this.streams[p.StreamID]
            if ok {
                session, err := s.setClient(netip.AddrPortFrom(msg.Addr().Addr(), p.Port))
                if err == nil {
                    utils.Warn(msg.SendResponse(packet.StreamResponse{StreamID: p.StreamID, RequestID: p.RequestID, SDP: session}))
                    return true
                } else {
                    slog.Error("Error setting client for", "stream", s.streamID, "err", err)
                }
            }

            utils.Warn(msg.SendResponse(packet.StreamEnd{StreamID: p.StreamID}))
            utils.Warn(msg.CloseConn())

            return true

        case packet.StreamCancel:
            p := msg.Packet().(packet.StreamCancel)

            s, ok := this.streams[p.StreamID]
            if !ok {
                slog.Warn("StreamCancel: inexistent streamID")
                return true
            }

            if msg.Addr().Addr() != s.client.Addr() {
                client := netip.AddrPortFrom(msg.Addr().Addr(), p.Port)
                slog.Warn("Invalid StreamCancel: client not registered with given streamID", "addr", client, "streamID", p.StreamID)
                return true
            }

            s.removeClient()
            return true
        }

    case service.TCPDisconnected:
        disc := sig.(service.TCPDisconnected)
        for _,s := range this.streams {
            if s.client.Addr() == disc.Addr().Addr() {
                s.removeClient()
            }
        }
        return true

    case service.UDPMessage:
        msg := sig.(service.UDPMessage)
        ping, ok := msg.Packet().(packet.Ping)
        if !ok { return false }
    
        utils.Warn(msg.SendResponse(ping))
        return true
    }

	return false
}

func main() {
    utils.SetupLogging()

    if len(os.Args) != 3 {
        fmt.Println("Usage: server <port> <config>")
        return
    }

    aux, err := strconv.ParseUint(os.Args[1], 10, 16)
    if err != nil {
        fmt.Println("Invalid port: the port must be an integer between 0 and 65535")
        return
    }
    port = uint16(aux)

    server := server{streams: make(map[string]*stream)}
    for streamID, filepath := range MustReadConfig(os.Args[2]) {
        metadata, err := start(streamID, filepath, true)
        
        if err != nil {
            fmt.Printf("Error loading stream '%s': %s\n", streamID, err)
        } else {
            server.streams[streamID] = metadata
            fmt.Printf("Hosting stream '%s'\n", streamID)
        }
    }

    serv.AddHandler(&server)
    err = serv.Run(&port, &port)
    if err != nil {
        slog.Error("Error running service", "err", err)
    }
}