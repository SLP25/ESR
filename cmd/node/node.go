package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/netip"
	"os"
	"strconv"
	"time"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)


type positiveResponse struct {
    from netip.Addr
    stream utils.StreamMetadata
}

type waitingStream struct {
    to utils.Set[netip.AddrPort]
    localPort uint16
}


var tcpPort uint16
var bootAddr netip.AddrPort
var serv service.Service

type node struct {
    neighbours []netip.AddrPort //TODO: differenciate actual neighbours from supposed neighbours
    servers []netip.AddrPort    //TODO: same for the servers
    probeRequests utils.Set[int]                //TODO: erase after a while
    probeResponses map[int]positiveResponse     //      same here (BUT! cant delete if there is a running/waiting stream)
    runningStreams streams
    waitingStreams map[string]waitingStream
}

func (this *node) isRP() bool {
    return len(this.servers) != 0
}

//Probes all servers and waits for their response.
//Returns the positive response from the server with the best connection metrics
func (this *node) probeServers(req packet.ProbeRequest) (packet.ProbeResponse, netip.Addr) {
    //TODO: make requests in parallel
    for _, s := range this.servers {
        var ans <-chan service.Signal

        serv.PauseHandleWhile(func() {
            serv.TCPServer().Send(req, s.Addr())
            ans = service.Intercept(&serv, func(sig service.Signal) bool {
                msg, ok := sig.(service.TCPMessage)
                if !ok { return false }
                
                resp, ok := msg.Packet().(packet.ProbeResponse)
                if !ok { return false }

                return msg.Addr().Addr() == s.Addr() && resp.RequestID == req.RequestID
            }, 1)
        })
        
        //TODO: compare metrics of multiple servers instead of picking just one
        resp := (<-ans).(service.TCPMessage).Packet().(packet.ProbeResponse)
        if resp.Exists {
            return resp, s.Addr()
        }
    }

    return req.RespondNonExistant(), netip.IPv4Unspecified()
}

func (this *node) fitsAditional(throughput int, addr netip.Addr) bool {
    return true //TODO: this.runningStreams.connUsage(addr) + throughput < ...
}


func (this *node) propagateProbeRequest(req packet.ProbeRequest, ignore ...netip.Addr) {
    for _, n := range this.neighbours {
        if !utils.Contains(ignore, n.Addr()) {
            utils.Warn(serv.TCPServer().Send(req, n.Addr()))
        }
    }
}

func (this *node) propagateProbeResponse(resp packet.ProbeResponse, ignore ...netip.Addr) {
    for _, n := range this.neighbours {
        if !utils.Contains(ignore, n.Addr()) && this.fitsAditional(resp.Stream.Throughput, n.Addr()) {
            utils.Warn(serv.TCPServer().Send(resp, n.Addr()))
        }
    }
}

func (this *node) cancelStream(streamID string, addr netip.Addr, port uint16) {
    if waitingStream, ok := this.waitingStreams[streamID]; ok {
        waitingStream.to.Remove(netip.AddrPortFrom(addr, port))
        if waitingStream.to.Length() == 0 {
            delete(this.waitingStreams, streamID)
        }
    }
    
    if this.runningStreams.removeSubscriber(streamID, addr, port) {
        addr := this.runningStreams.endSubscription(streamID)
        p := packet.StreamCancel{StreamID: streamID, Port: tcpPort}
        utils.Warn(serv.TCPServer().Send(p, addr))
    }
}

//If the requestID is already in use, the request is ignored
//If there is a running stream, a response is deduced and handled
//Otherwise, the request is propagated to both neighbours and servers. The servers' response is then handled
func (this *node) handleProbeRequest(req packet.ProbeRequest, source netip.Addr) {
    if this.probeRequests.Contains(req.RequestID) {
        return
    }

    this.probeRequests.Add(req.RequestID)

    if stream, ok := this.runningStreams[req.StreamID]; ok {
        this.handleProbeResponse(req.RespondExistant(stream.metadata), stream.from)
    } else {
        this.propagateProbeRequest(req, source)

        resp, s := this.probeServers(req)
        if resp.Exists || this.isRP() {
            this.handleProbeResponse(resp, s)
        }
    }
}

//If a response is already registered for this requestID, this response is ignored
//The response is stored and progagated.
//Then, if there is a correspondent waiting stream, a StreamRequest is sent to this response's address
func (this *node) handleProbeResponse(resp packet.ProbeResponse, source netip.Addr) {
    this.probeRequests.Add(resp.RequestID)

    if _, ok := this.probeResponses[resp.RequestID]; ok {
        return
    }

    if resp.Exists {
        this.probeResponses[resp.RequestID] = positiveResponse{from: source, stream: resp.Stream}
    }
    
    this.propagateProbeResponse(resp, source)

    if waitingStream, ok := this.waitingStreams[resp.StreamID]; ok {
        if !resp.Exists { //we don't want to start a probe request if the stream doesn't exist
            for addrport := range waitingStream.to {
                utils.Warn(serv.TCPServer().Send(packet.StreamEnd{StreamID: resp.StreamID}, addrport.Addr()))
            }
            delete(this.waitingStreams, resp.StreamID)
        } else {
            //receive "ghost" StreamRequest from all subscribers to the stream to propagate it upwards
            this.handleStreamRequest(resp.StreamID, resp.RequestID, waitingStream.to.ToSlice()...)
        }

        delete(this.waitingStreams, resp.StreamID)
    }
}


func (this *node) handleStreamRequest(streamID string, requestID int, dests ...netip.AddrPort) {
    if len(dests) == 0 {
        slog.Warn("handleStreamRequest: called with no dests")
        return
    }
    
    if resp, ok := this.probeResponses[requestID]; ok {

        if s, ok := this.runningStreams[streamID]; ok {
            for _, addrport := range dests {
                s.to.Add(addrport)
                serv.TCPServer().Send(packet.StreamResponse{SDP: s.sdp}, addrport.Addr())
            }            
        } else if w, ok := this.waitingStreams[streamID]; ok {
            for _, addrport := range dests {
                w.to.Add(addrport)
            }
        } else {
            var port uint16
            utils.Warn(serv.AddUDPServer(&port))

            p := packet.StreamRequest{StreamID: streamID, RequestID: requestID, Port: port}
            err := serv.TCPServer().Send(p, resp.from)
            if err != nil {
                slog.Error("Unable to propagate StreamRequest", "err", err)
                return
            }

            this.waitingStreams[streamID] = waitingStream{to: utils.SetFrom[netip.AddrPort](dests...), localPort: port}
        }
        
    } else if !this.probeRequests.Contains(requestID) {
        for _, addrPort := range dests {
            if _, ok := this.waitingStreams[streamID]; !ok {
                this.waitingStreams[streamID] = waitingStream{to: utils.EmptySet[netip.AddrPort]()}
            }
            
            this.waitingStreams[streamID].to.Add(addrPort)
        }
        
        req := packet.ProbeRequest{StreamID: streamID, RequestID: requestID}
        this.handleProbeRequest(req, netip.IPv4Unspecified())
    }
}


func (this *node) Handle(sig service.Signal) bool {
    switch sig.(type) {
    case service.Init:
        request := packet.StartupRequest{Service: utils.Node}
        response, err := service.InterceptTCPResponse[packet.StartupResponseNode](&serv, request, bootAddr)
        if err != nil {
            slog.Error("Error on Init:", "err", err)
            serv.Close()
            return true
        }

        utils.Warn(serv.TCPServer().CloseConn(bootAddr.Addr()))

        this.neighbours = response.Neighbours
        for _, n := range response.Neighbours {
            err := serv.TCPServer().Connect(n)

            if err != nil {
                slog.Warn("Unable to connect to neighbour node", "err", err)
                //TODO: retry? or wait for them to initiate?
            }
        }

        this.servers = response.Servers
        for _, s := range response.Servers {
            err := serv.TCPServer().Connect(s)

            if err != nil {
                slog.Error("Unable to connect to server", "err", err)
                //TODO: retry connection every once in a while
            }
        }

        return true

    case service.TCPDisconnected:
        disc := sig.(service.TCPDisconnected)
        sources, dests := this.runningStreams.eraseAddr(disc.Addr().Addr())

        //cancel unused stream
        for _, streamID := range dests {
            addr := this.runningStreams.endSubscription(streamID)
            p := packet.StreamCancel{StreamID: streamID, Port: tcpPort}
            utils.Warn(serv.TCPServer().Send(p, addr))
        }

        //re-request unavailable streams
        for streamID, addrPorts := range sources {
            randInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
            //we use goroutines in order to run all requests in parallel (TODO: controlo de concorrencia)
            go this.handleStreamRequest(streamID, randInt, addrPorts...)
        }

        return true

    case service.TCPMessage:
        msg := sig.(service.TCPMessage)
        switch msg.Packet().(type) {

        case packet.ProbeRequest:
            req := msg.Packet().(packet.ProbeRequest)
            this.handleProbeRequest(req, msg.Addr().Addr())
            return true

        case packet.ProbeResponse:
            resp := msg.Packet().(packet.ProbeResponse)
            this.handleProbeResponse(resp, msg.Addr().Addr())
            return true

        case packet.StreamRequest:
            p := msg.Packet().(packet.StreamRequest)
            this.handleStreamRequest(p.StreamID, p.RequestID, netip.AddrPortFrom(msg.Addr().Addr(), p.Port))
            return true

        case packet.StreamResponse:
            p := msg.Packet().(packet.StreamResponse)

            if resp, ok := this.probeResponses[p.RequestID]; ok {
                if w, ok := this.waitingStreams[p.StreamID]; ok {
                    this.runningStreams.startSubscription(p.StreamID, resp, w.localPort, p.SDP, w.to.ToSlice())
                    for addrport := range w.to {
                        utils.Warn(serv.TCPServer().Send(packet.StreamResponse{}, addrport.Addr()))
                    }
                }
            }

            return true

        case packet.StreamCancel:
            p := msg.Packet().(packet.StreamCancel)
            this.cancelStream(p.StreamID, msg.Addr().Addr(), p.Port)
            return true

        case packet.StreamEnd:
            p := msg.Packet().(packet.StreamEnd)

            if this.runningStreams[p.StreamID].from != msg.Addr().Addr() { //discard
                return true
            }

            //propagate StreamEnd
            for addr := range this.runningStreams[p.StreamID].to {
                utils.Warn(serv.TCPServer().Send(p, addr.Addr()))
            }

            //locally remove the subscription
            delete(this.waitingStreams, p.StreamID)
            this.runningStreams.endSubscription(p.StreamID)

            return true
        }

    case service.UDPMessage:
        msg := sig.(service.UDPMessage)
        switch msg.Packet().(type) {

        case packet.StreamPacket:
            p := msg.Packet().(packet.StreamPacket)

            for addr := range this.runningStreams[p.StreamID].to { //TODO: fix crash if not initialized (see if this could happen elsewhere)
                utils.Warn(service.SendUDP(p, addr))
            }
            return true
        }
    }

    return false
}

func main() {
    utils.SetupLogging()

    if len(os.Args) != 3 {
        fmt.Println("Usage: node <port> <bootAddr>")
        return
    }

    aux, err := strconv.ParseUint(os.Args[1], 10, 16)
    if err != nil {
        fmt.Println("Invalid port: the port must be an integer between 0 and 65535")
        return
    }
    tcpPort = uint16(aux)

    bootAddr, err = netip.ParseAddrPort(os.Args[2])
    if err != nil {
        fmt.Println("Invalid boot address:", err)
        return
    }

    node := node{
        probeRequests: utils.EmptySet[int](),
        probeResponses: make(map[int]positiveResponse),
        runningStreams: make(streams),
        waitingStreams: make(map[string]utils.Set[netip.AddrPort]),
    }
    serv.AddHandler(&node)
    
    err = serv.Run(&tcpPort)
    if err != nil {
        slog.Error("Error running service", "err", err)
    }
}