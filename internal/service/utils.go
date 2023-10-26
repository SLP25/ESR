package service

import (
	"net/netip"
)

type addr struct {
	network string
	addrport netip.AddrPort
}

func (this addr) Network() string {
	return this.network
}

func (this addr) String() string {
	return this.addrport.String()
}