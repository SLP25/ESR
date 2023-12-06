package utils

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"strconv"
	"time"
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

func RandID() uint32 {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Uint32()
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

func IPPortToInt(ip netip.AddrPort) uint32 {
	bytes := ip.Addr().As4()
	return binary.BigEndian.Uint32(bytes[0:4])
}

func AbsDiff(val1 uint32, val2 uint32) uint32 {
	if val1 < val2 {
		return val2 - val1
	} else {
		return val1 - val2
	}
}
