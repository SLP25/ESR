package packet

import (
	"net/netip"

	"github.com/SLP25/ESR/internal/utils"
)

//any -> bootstrapper
type StartupRequest struct {
	Service utils.ServiceType
}

//bootstrapper -> any
type StartupResponse struct {
	ConnectTo netip.AddrPort
}

type Ping struct {

}

type Pong struct {
	neighbours map[netip.AddrPort] utils.Metrics
}