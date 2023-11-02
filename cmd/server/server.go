package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/SLP25/ESR/internal/service"
)


type server struct {
    test int
}

var serv service.Service


func (this *server) Handle(sig service.Signal) bool {
	return false
}

func main() {
    fmt.Println("Hello! I'm the server")

	handler := slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
    log := slog.New(slog.NewTextHandler(os.Stdout, &handler))
    slog.SetDefault(log)

    server := server{}
    serv.AddHandler(&server)
    
    err := serv.Run(4003, 4003)
    if err != nil {
        slog.Error("Error running service:", err)
    }
}