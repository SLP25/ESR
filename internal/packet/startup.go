package packet

import (
	"net/netip"

	"github.com/SLP25/ESR/internal/utils"
)

//any -> bootstrapper
type StartupRequest struct {
	Service utils.ServiceType
}

//bootstrapper -> client
type StartupResponseClient struct {
	ConnectTo netip.AddrPort
}

//bootstrapper -> node
type StartupResponseNode struct {
	Neighbours map[netip.AddrPort]utils.Metrics
	Servers []netip.AddrPort
}


type Ping struct {
	ID uint32
	//Data []byte 		//to measure error rate (not implemented)
}