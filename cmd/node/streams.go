package main

import (
	"log/slog"
	"net/netip"

	"github.com/SLP25/ESR/internal/utils"
	"github.com/pion/sdp/v2"
)


type stream struct {
    from netip.Addr
    toLocal uint16
    to utils.Set[netip.AddrPort]
    metadata utils.StreamMetadata
    sdp sdp.SessionDescription
}


type streams map[string]*stream

func (this streams) startSubscription(streamID string, resp positiveResponse, port uint16, sdp sdp.SessionDescription, children []netip.AddrPort) {
    if _, ok := this[streamID]; ok {
        slog.Error("startSubscription: called on existing streamID", "streamID", streamID)
        return
    }

    this[streamID] = &stream{
        from: resp.from,
        toLocal: port,
        to: utils.SetFrom(children...),
        metadata: resp.stream,
        sdp: sdp,
    }
}

func (this streams) endSubscription(streamID string) netip.Addr {
    addr := this[streamID].from
    delete(this, streamID)
    return addr
}

//returns true if the addr was the last subscriber for that stream
func (this streams) removeSubscriber(streamID string, addr netip.Addr, port uint16) bool {
    if _, ok := this[streamID]; !ok {
        return false
    }
    
    return this[streamID].to.Remove(netip.AddrPortFrom(addr, port)) && this[streamID].to.Length() == 0
}

//returns the streams where the addr was the source and the streamIDs which became empty
func (this streams) eraseAddr(addr netip.Addr) (map[string][]netip.AddrPort, []string) {
    fromSubs := make(map[string][]netip.AddrPort)
    emptyToSubs := make([]string, 0)
    
    for streamID, stream := range this {
        if stream.from == addr {
            fromSubs[streamID] = utils.GetKeys(stream.to)
            delete(this, streamID)
        } else {
            for addrport := range stream.to {
                if addrport.Addr() == addr && this.removeSubscriber(streamID, addrport.Addr(), addrport.Port()) {
                    emptyToSubs = append(emptyToSubs, streamID)
                }
            }
        }
    }

    return fromSubs, emptyToSubs
}

//Returns the sums of the throughput of all streams running to and from the given address
func (this streams) connUsage(addr netip.Addr) int {
    total := 0

    for _, stream := range this {
        if addr == stream.from {
            total += stream.metadata.Throughput
        }

        for addrport := range stream.to {
            if addrport.Addr() == addr {
                total += stream.metadata.Throughput
            }
        }
    }

    return total
}