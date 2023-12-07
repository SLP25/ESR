package main

import (
	"log/slog"
	"net/netip"
	"os"
	"strconv"
    "runtime"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)


type probeResponse struct {
    from netip.Addr
    stream *utils.StreamMetadata
}

type neighbourInfo struct {
    port uint16
    metrics utils.Metrics
}

var tcpPort uint16
var bootAddr netip.AddrPort
var serv service.Service

type node struct {
    neighbours map[netip.Addr]neighbourInfo
    servers []netip.AddrPort
    probeRequests utils.Set[uint32]                //TODO: erase after a while
    probeResponses map[uint32]probeResponse     //      same here (BUT! cant delete if there is a running/waiting stream)
    runningStreams streams                      //This node is currently receiving and sending packets for these streams
    waitingStreams map[string]*waitingStream     //This node is currently waiting for a StreamResponse for these streams
}

func (this *node) isRP() bool {
    return len(this.servers) != 0
}

//Probes all servers and waits for their response.
//Returns the positive response from the server with the best connection metrics
func (this *node) probeServers(req packet.ProbeRequest) (packet.ProbeResponse, netip.Addr) {
    answers := make(map[netip.AddrPort]<-chan service.Signal)
    for _, s := range this.servers {
        serv.PauseHandleWhile(func() {
            err := serv.TCPServer().SendConnect(req, s)
            if err != nil {
                slog.Warn("Unable to connect to server", "addr", s, "err", err)
                return
            }
            st := s
            answers[st] = service.Intercept(&serv, func(sig service.Signal) bool {
                msg, ok := sig.(service.TCPMessage)
                if !ok { return false }
                
                resp, ok := msg.Packet().(packet.ProbeResponse)
                if !ok { return false }

                return msg.Addr().Addr() == st.Addr() && resp.RequestID == req.RequestID
            }, 1)
        })
    }

    bestServer := netip.AddrPortFrom(netip.IPv4Unspecified(), 0)
    bestResponse := req.RespondNonExistant()

    for s, c := range answers {
        resp := (<-c).(service.TCPMessage).Packet().(packet.ProbeResponse)

        if resp.Exists {
            //TODO: compare metrics
            //if this.metrics[s].BetterThan(this.metrics[bestServer]) {
            //    bestServer = s
            //    bestResponse = resp
            //}
            return resp, s.Addr()
        }
    }

    return bestResponse, bestServer.Addr()
}

func (this *node) fitsAditional(bitrate int, addr netip.Addr) bool { //TODO: also look at waitingStreams?
    return this.runningStreams.connUsage(addr) + bitrate < this.neighbours[addr].metrics.Bandwidth
}


func (this *node) propagateProbeRequest(req packet.ProbeRequest, ignore ...netip.Addr) {
    for addr, ni := range this.neighbours {
        if !utils.Contains(ignore, addr) {
            utils.Warn(serv.TCPServer().SendConnect(req, netip.AddrPortFrom(addr, ni.port)))
        }
    }
}

func (this *node) propagateProbeResponse(resp packet.ProbeResponse, ignore ...netip.Addr) {
    for addr, ni := range this.neighbours {
        if !utils.Contains(ignore, addr) && this.fitsAditional(resp.Stream.Bitrate, addr) {
            utils.Warn(serv.TCPServer().SendConnect(resp, netip.AddrPortFrom(addr, ni.port)))
        }
    }
}

func (this *node) cancelStream(streamID string, addr netip.Addr, port uint16) {
    if waitingStream, ok := this.waitingStreams[streamID]; ok {
        waitingStream.to.Remove(netip.AddrPortFrom(addr, port))
        if waitingStream.to.Length() == 0 {
            //fmt.Println("Canceling waiting stream")
            if waitingStream.localPort != 0 {
                serv.RemoveUDPServer(waitingStream.localPort)
            }
            delete(this.waitingStreams, streamID)
        }
    }
    
    if this.runningStreams.removeSubscriber(streamID, addr, port) {
        //fmt.Println("Canceling running stream (sending StreamCancel)")
        addr, localPort := this.runningStreams.endSubscription(streamID)
        serv.RemoveUDPServer(localPort)
        p := packet.StreamCancel{StreamID: streamID, Port: localPort}
        utils.Warn(serv.TCPServer().Send(p, addr))
    }
}

//If the requestID is already in use, the request is ignored
//If there is a running stream, a response is deduced and handled
//Otherwise, the request is propagated to both neighbours and servers. The servers' response is then handled
func (this *node) handleProbeRequest(req packet.ProbeRequest, source netip.Addr) {
    //fmt.Println("Processing probe request")
    
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
    //fmt.Println("Processing probe response")
    
    this.probeRequests.Add(resp.RequestID)

    if _, ok := this.probeResponses[resp.RequestID]; ok {
        return
    }

    if resp.Exists {
        this.probeResponses[resp.RequestID] = probeResponse{from: source, stream: &resp.Stream}
    } else {
        this.probeResponses[resp.RequestID] = probeResponse{from: source, stream: nil}
    }
    
    this.propagateProbeResponse(resp, source)

    if waitingStream, ok := this.waitingStreams[resp.StreamID]; ok {
        //fmt.Println("ProbeResponse reaction")

        if !resp.Exists { //we don't want to start a probe request if the stream doesn't exist
            for addrport := range waitingStream.to {
                utils.Warn(serv.TCPServer().Send(packet.StreamEnd{StreamID: resp.StreamID}, addrport.Addr()))
            }
            delete(this.waitingStreams, resp.StreamID)
        } else {
            //receive "ghost" StreamRequest from all subscribers to the stream to propagate it upwards
            this.handleStreamRequest(resp.StreamID, resp.RequestID, waitingStream.to.ToSlice()...)
        }
    }
}


func (this *node) handleStreamRequest(streamID string, requestID uint32, dests ...netip.AddrPort) {    
    //fmt.Println("Processing stream request")
    
    if len(dests) == 0 {
        slog.Warn("handleStreamRequest: called with no dests")
        return
    }
    
    if s, ok := this.runningStreams[streamID]; ok {
        for _, addrport := range dests {
            s.to.Add(addrport)
            utils.Warn(serv.TCPServer().Send(packet.StreamResponse{SDP: s.sdp,StreamID:streamID,RequestID:requestID}, addrport.Addr()))
        }
    } else if resp, ok := this.probeResponses[requestID]; ok {
        if resp.stream == nil {
            for _, addrport := range dests {
                s.to.Add(addrport)
                utils.Warn(serv.TCPServer().Send(packet.StreamEnd{StreamID: streamID}, addrport.Addr()))
            }
        } else {
            //fmt.Println("Add addrport to waitingStreams")

            if _, ok := this.waitingStreams[streamID]; !ok {
                this.waitingStreams[streamID] = &waitingStream{to: utils.EmptySet[netip.AddrPort](), localPort: 0}
            }

            for _, addrport := range dests {
                this.waitingStreams[streamID].to.Add(addrport)
            }

            if this.waitingStreams[streamID].localPort == 0 {

                //fmt.Println("Open a new port and send StreamRequest")

                var port uint16
                serv.AddUDPServer(&port)
                this.waitingStreams[streamID].localPort = port

                p := packet.StreamRequest{StreamID: streamID, RequestID: requestID, Port: port}
                err := serv.TCPServer().Send(p, resp.from)
                if err != nil {
                    slog.Error("Unable to propagate StreamRequest", "err", err)
                    return
                }
            }
        }
    } else if !this.probeRequests.Contains(requestID) {
        //fmt.Println("Add dests to waitingStreams and send probeRequest")
        for _, addrPort := range dests {
            if _, ok := this.waitingStreams[streamID]; !ok {
                this.waitingStreams[streamID] = &waitingStream{to: utils.EmptySet[netip.AddrPort]()}
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

        this.neighbours = make(map[netip.Addr]neighbourInfo)
        for n, m := range response.Neighbours {
            this.neighbours[n.Addr()] = neighbourInfo{port: n.Port(), metrics: m}

            err := serv.TCPServer().Connect(n)
            if err != nil {
                slog.Warn("Unable to connect to neighbour node", "err", err)
                //TODO: retry? or wait for them to initiate?
            }
        }

        this.servers = response.Servers
        return true

    case service.TCPDisconnected:
        disc := sig.(service.TCPDisconnected)
        sources, dests := this.runningStreams.eraseAddr(disc.Addr().Addr())

        //cancel unused stream
        for streamID, port := range dests {
            //fmt.Println("Canceling unused stream")

            addr, _ := this.runningStreams.endSubscription(streamID)
            p := packet.StreamCancel{StreamID: streamID, Port: port}
            utils.Warn(serv.TCPServer().Send(p, addr))
        }

        //re-request unavailable streams
        for streamID, waiting := range sources {
            //fmt.Println("Re-request unavailable stream")

            this.waitingStreams[streamID] = &waiting
            randInt := utils.RandID()
            //we use goroutines in order to run all requests in parallel (TODO: controlo de concorrencia)
            go this.handleStreamRequest(streamID, randInt, waiting.to.ToSlice()...)
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

            //fmt.Println("Processing StreamResponse", p)

            if resp, ok := this.probeResponses[p.RequestID]; ok {
                if resp.stream == nil {
                    slog.Warn("Received StreamResponse for non-existant stream", "streamID", p.StreamID, "requestID", p.RequestID)
                } else if w, ok := this.waitingStreams[p.StreamID]; ok && w.localPort != 0 {
                    //fmt.Println("Adding stream to runningStreams and removing from waitingStreams", w)
                    this.runningStreams.startSubscription(p.StreamID, resp, w.localPort, p.SDP, w.to.ToSlice())
                    for addrport := range w.to {
                        utils.Warn(serv.TCPServer().Send(p, addrport.Addr()))
                    }
                    delete(this.waitingStreams, p.StreamID)

                    //fmt.Println("runningStreams:", this.runningStreams)
                    //fmt.Println("waitingStreams:", this.waitingStreams)
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
            serv.RemoveUDPServer(this.waitingStreams[p.StreamID].localPort)
            delete(this.waitingStreams, p.StreamID)
            this.runningStreams.endSubscription(p.StreamID)

            return true
        }

    case service.UDPMessage:
        msg := sig.(service.UDPMessage)
        switch msg.Packet().(type) {

        case packet.StreamPacket:
            p := msg.Packet().(packet.StreamPacket)

            for _, addrport := range this.runningStreams.ForwardAddresses(msg.LocalPort()) {
                utils.Warn(service.SendUDP(p, addrport))
            }

            return true
        }
    }

    return false
}

func main() {
    runtime.GOMAXPROCS(1)
    utils.SetupLogging()

    if len(os.Args) != 3 {
        //fmt.Println("Usage: node <port> <bootAddr>")
        return
    }

    aux, err := strconv.ParseUint(os.Args[1], 10, 16)
    if err != nil {
        //fmt.Println("Invalid port: the port must be an integer between 0 and 65535")
        return
    }
    tcpPort = uint16(aux)

    bootAddr, err = netip.ParseAddrPort(os.Args[2])
    if err != nil {
        //fmt.Println("Invalid boot address:", err)
        return
    }

    node := node{
        probeRequests: utils.EmptySet[uint32](),
        probeResponses: make(map[uint32]probeResponse),
        runningStreams: make(streams),
        waitingStreams: make(map[string]*waitingStream),
    }
    serv.AddHandler(&node)
    
    err = serv.Run(&tcpPort)
    if err != nil {
        slog.Error("Error running service", "err", err)
    }
}