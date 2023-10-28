package main

import (
	"fmt"
	"net/netip"

	"github.com/SLP25/ESR/internal/service"
)

var serv service.Service

type client struct {
    accessNode netip.Addr
}

func (this client) Handle(sig service.Signal) bool {
    switch sig.(type) {
    case service.Init:
        fmt.Println("Ready!")
        //TODO: connect to bootstrapper
        //this.accessNode = ...

    case service.Message:
        fmt.Println("Received packet")
        //TODO: handle packet

    default:
        return false
    }

    return true
}

func main() {
    fmt.Println("Hello! I'm the client")

    client := client{}
    serv.AddHandler(client)
    serv.Run(69, 69)
    
    fmt.Println("Bye!")
}