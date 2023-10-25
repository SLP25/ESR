package packet

import (
	"net"

	"github.com/SLP25/ESR/internal/utils"
)

//any -> bootstrapper
type StartupRequest struct {
	service utils.ServiceType
}

//bootstrapper -> any
type StartupResponse struct {
	RP net.Addr
}