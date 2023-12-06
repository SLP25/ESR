package packet

import "github.com/SLP25/ESR/internal/utils"

//node -> node
type ProbeRequest struct {
	StreamID string
	RequestID uint32 //random number to identify a request
}

//node -> node
type ProbeResponse struct {
	StreamID string
	RequestID uint32 //random number to identify a request
	Exists bool
	Stream utils.StreamMetadata
}


func (this ProbeRequest) RespondNonExistant() ProbeResponse {
	return ProbeResponse{StreamID: this.StreamID, RequestID: this.RequestID, Exists: false}
}

func (this ProbeRequest) RespondExistant(stream utils.StreamMetadata) ProbeResponse {
	return ProbeResponse{StreamID: this.StreamID, RequestID: this.RequestID, Exists: true, Stream: stream}
}