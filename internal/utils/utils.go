package utils

import (
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"strconv"
)

type ServiceType byte

const (
	Bootstrapper ServiceType = iota
	Client
	Node
	Server
)

func SetupLogging() {
	handler := slog.HandlerOptions{AddSource: false, Level: slog.LevelDebug}
    log := slog.New(slog.NewTextHandler(os.Stdout, &handler))
    slog.SetDefault(log)
}

func MatchesPort(pattern uint16, val uint16) bool {
	return pattern == 0 || pattern == val
}

func Matches(pattern netip.AddrPort, val netip.AddrPort) bool {
	return pattern.Addr() == val.Addr() && (pattern.Port() == 0 || pattern.Port() == val.Port())
}


func Contains[T comparable](list []T, item T) bool {
	for _, i := range list {
		if item == i {
			return true
		}
	}

	return false
}


func Ellipsis(val any, maxLen int) string {
	s := fmt.Sprint(val)
	if len(s) <= maxLen {
		return s
	} else {
		return s[:maxLen-3] + "..."
	}
}

func FindFreePorts(n int) []uint16 {
	ans := make([]uint16, n)
	conns := make([]net.PacketConn, n)

	for i := 0; i < n; i++ {
		var err error
		conns[i], err = net.ListenPacket("udp", ":0")
		if err != nil {
			panic("Couldn't find " + strconv.Itoa(n) + " open ports: " + err.Error())
		}
		ans[i] = netip.MustParseAddrPort(conns[i].LocalAddr().String()).Port()
	}

	for _, c := range conns {
		c.Close()
	}
	
	return ans
}