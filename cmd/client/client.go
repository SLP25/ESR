package main

import (
	"fmt"
	"net"

	"github.com/SLP25/ESR/internal/service"
)

var serv service.Service

type client struct {
    accessNode net.Addr
}

func (this client) Handle(sig service.Signal) {
    switch sig.(type) {
    case service.Init:
        fmt.Println("Ready!")
        //TODO: connect to bootstrapper
        //this.accessNode = ...

    case service.Message:
        fmt.Println("Received packet")
        //TODO: handle packet
    }
}

func main() {
    fmt.Println("Hello! I'm the client")

    client := client{}
    serv = service.Service{Handler: client}
    serv.Run(69, 69)
    
    fmt.Println("Bye!")
}