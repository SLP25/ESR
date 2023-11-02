package main

import (
	"fmt"
	"log/slog"
	"net/netip"
	"os"

	"github.com/SLP25/ESR/internal/service"
)

var serv service.Service

type client struct {
    accessNode netip.Addr
}

func (this client) Handle(sig service.Signal) bool {
    switch sig.(type) {
    case service.Init:
        //TODO: connect to bootstrapper
        //this.accessNode = ...

    case service.Message:
        //TODO: handle packet

    default:
        return false
    }

    return true
}

func main() {
    fmt.Println("Hello! I'm the client")

    handler := slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
    log := slog.New(slog.NewTextHandler(os.Stdout, &handler))
    slog.SetDefault(log)

    client := client{}
    serv.AddHandler(client)

    err := serv.Run(69, 69)
    if err != nil {
        slog.Error("Error running service", err)
    }
}