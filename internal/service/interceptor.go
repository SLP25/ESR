package service

import (
	"fmt"
	"log/slog"
	"net/netip"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/utils"
)

type checker func(Signal) bool

type interceptor struct {
	service *Service
	accept checker
	n int
	ans chan Signal
}

func (this *interceptor) Handle(sig Signal) bool {
	if this.accept(sig) {

		if sig == nil {
			fmt.Println("BATATA")
		}

		this.n -= 1
		if this.n == 0 {
			this.service.RemoveHandler(this)
		}

		select {
		case this.ans <- sig:
			if this.n == 0 {
				close(this.ans)
			}
			return true
		default:
			slog.Warn("Interceptor hit its maximum buffer size. Closing...")
			close(this.ans)
			return false
		}
	}

	return false
}

// Intercepts the first n signals that satisfy the condition
// If n is non-positive, all signals that satisfy the condition are intercepted
func Intercept(service *Service, accept checker, n int) <-chan Signal {
	i := &interceptor{service: service, ans: make(chan Signal, min(n, 20)), accept: accept, n: n}
	service.AddHandler(i)
	return i.ans
}

// Intercepts the first n signals of the specified type
// If n is non-positive, all signals of the specified type are intercepted
func InterceptSignal[T Signal](service *Service, n int) <-chan T {
	return utils.CastChan[Signal, T](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case T:
			return true
		default:
			return false
		}
	}, n))
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptMessages(service *Service, addr netip.AddrPort, n int) <-chan Message {
	return utils.CastChan[Signal, Message](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case Message:
			return sig.(Message).Addr() == addr
		default:
			return false
		}
	}, n))
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptTCPMessages(service *Service, addr netip.AddrPort, n int) <-chan TCPMessage {
	return utils.CastChan[Signal, TCPMessage](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case TCPMessage:
			return sig.(TCPMessage).Addr() == addr
		default:
			return false
		}
	}, n))
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptUDPMessages(service *Service, addr netip.AddrPort, n int) <-chan TCPMessage {
	return utils.CastChan[Signal, TCPMessage](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case TCPMessage:
			return sig.(TCPMessage).Addr() == addr
		default:
			return false
		}
	}, n))
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptPackets[T packet.Packet](service *Service, addr netip.AddrPort, n int) <-chan T {
	return utils.MapChan[Signal, T](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case Message:
			msg := sig.(Message)
			switch msg.Packet().(type) {
			case T:
				return msg.Addr() == addr
			}
		}
		return false
	}, n), func (sig Signal) T {
		return sig.(Message).Packet().(T)
	})
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptTCPPackets[T packet.Packet](service *Service, addr netip.AddrPort, n int) <-chan T {
	return utils.MapChan[Signal, T](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case TCPMessage:
			msg := sig.(TCPMessage)
			switch msg.Packet().(type) {
			case T:
				return msg.Addr() == addr
			}
		}
		return false
	}, n), func (sig Signal) T {
		return sig.(TCPMessage).Packet().(T)
	})
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptUDPPackets[T packet.Packet](service *Service, addr netip.AddrPort, n int) <-chan T {
	return utils.MapChan[Signal, T](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case UDPMessage:
			msg := sig.(UDPMessage)
			switch msg.Packet().(type) {
			case T:
				return msg.Addr() == addr
			}
		}
		return false
	}, n), func (sig Signal) T {
		return sig.(UDPMessage).Packet().(T)
	})
}

//TODO: timeout