package utils

import (
	"log/slog"
	"net/netip"
	"os"
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