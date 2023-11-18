package packet

//node/client -> node/server
type StreamRequest struct {
	StreamID string
	RequestID int
	Port uint16
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


//server/node -> node/client
type StreamPacket struct {
	StreamID string
	Content []byte
}