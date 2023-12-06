package packet

import (
	"net/netip"

	"github.com/pion/sdp/v2"
)

//node/client -> node/server
type StreamRequest struct {
	StreamID string
	RequestID uint32
	Port uint16
}

type StreamResponse struct {
	StreamID string
	RequestID uint32
	SDP sdp.SessionDescription
}

//node/client -> node/server
type StreamCancel struct {
	StreamID string
	Port uint16
}

//server/node -> node/client
type StreamEnd struct {
	StreamID string
}

type StreamType byte

const (
	Video StreamType = iota
	Audio
	VideoControl
	AudioControl
)

//server/node -> node/client
type StreamPacket struct {
	Type StreamType
	Content []byte
}

func (this *StreamResponse) SetOrigin(origin netip.AddrPort) {
	this.SDP.Origin.UnicastAddress = origin.String()
}

func (this *StreamResponse) SetPorts(video uint16, audio uint16) {
	for _, m := range this.SDP.MediaDescriptions {
		if m.MediaName.Media == "video" {
			m.MediaName.Port = sdp.RangedPort{Value: int(video)}
		} else if m.MediaName.Media == "audio" {
			m.MediaName.Port = sdp.RangedPort{Value: int(audio)}
		}
	}
}