package main

import (
	"log/slog"
	"net/netip"

	"github.com/SLP25/ESR/internal/utils"
	"github.com/pion/sdp/v2"
)

type waitingStream struct {
    to utils.Set[netip.AddrPort]
    localPort uint16
}

type stream struct {
    from netip.Addr
    toLocal uint16
    to utils.Set[netip.AddrPort]
    metadata utils.StreamMetadata
    sdp sdp.SessionDescription
}

type streams map[string]*stream

func (this streams) startSubscription(streamID string, resp probeResponse, port uint16, sdp sdp.SessionDescription, children []netip.AddrPort) {
    if _, ok := this[streamID]; ok {
        slog.Error("startSubscription: called on existing streamID", "streamID", streamID)
        return
    }

    if resp.stream == nil {
        slog.Error("startSubscription: called with negative probe response", "streamID", streamID)
        return
    }

    this[streamID] = &stream{
        from: resp.from,
        toLocal: port,
        to: utils.SetFrom(children...),
        metadata: *resp.stream,
        sdp: sdp,
    }
}

func (this streams) ForwardAddresses(localPort uint16) []netip.AddrPort {
    for _, stream := range this {
        if localPort == stream.toLocal {
            return stream.to.ToSlice()
        }
    }
    return make([]netip.AddrPort, 0)
}

func (this streams) endSubscription(streamID string) (netip.Addr, uint16) {
    addr := this[streamID].from
    port := this[streamID].toLocal
    delete(this, streamID)
    return addr, port
}

//returns true if the addr was the last subscriber for that stream
func (this streams) removeSubscriber(streamID string, addr netip.Addr, port uint16) bool {
    if _, ok := this[streamID]; !ok {
        return false
    }
    
    return this[streamID].to.Remove(netip.AddrPortFrom(addr, port)) && this[streamID].to.Length() == 0
}

//returns the streams where the addr was the source and the streamIDs which became empty
func (this streams) eraseAddr(addr netip.Addr) (map[string]waitingStream, map[string]uint16) {
    fromSubs := make(map[string]waitingStream)
    emptyToSubs := make(map[string]uint16, 0)
    
    for streamID, stream := range this {
        if stream.from == addr {
            fromSubs[streamID] = waitingStream{to: stream.to, localPort: stream.toLocal} 
            delete(this, streamID)
        } else {
            for addrport := range stream.to {
                if addrport.Addr() == addr && this.removeSubscriber(streamID, addrport.Addr(), addrport.Port()) {
                    emptyToSubs[streamID] = stream.toLocal
                }
            }
        }
    }

    return fromSubs, emptyToSubs
}

//Returns the sums of the bitrate of all streams running to and from the given address
func (this streams) connUsage(addr netip.Addr) int {
    total := 0

    for _, stream := range this {
        if addr == stream.from {
            total += stream.metadata.Bitrate
        }

        for addrport := range stream.to {
            if addrport.Addr() == addr {
                total += stream.metadata.Bitrate
            }
        }
    }

    return total
}