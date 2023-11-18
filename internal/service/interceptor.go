package service

import (
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
			slog.Warn("Interceptor hit its maximum buffer size. Signal dropped")
			return true
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
		_, ok := sig.(T)
		return ok
	}, n))
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptMessages(service *Service, addr netip.AddrPort, n int) <-chan Message {
	return utils.CastChan[Signal, Message](Intercept(service, func(sig Signal) bool {
		msg, ok := sig.(Message)
		return ok && utils.Matches(addr, msg.Addr())
	}, n))
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptTCPMessages(service *Service, addr netip.AddrPort, n int) <-chan TCPMessage {
	return utils.CastChan[Signal, TCPMessage](Intercept(service, func(sig Signal) bool {
		msg, ok := sig.(TCPMessage)
		return ok && utils.Matches(addr, msg.Addr())
	}, n))
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptUDPMessages(service *Service, addr netip.AddrPort, n int) <-chan UDPMessage {
	return utils.CastChan[Signal, UDPMessage](Intercept(service, func(sig Signal) bool {
		msg, ok := sig.(UDPMessage)
		return ok && utils.Matches(addr, msg.Addr())
	}, n))
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptPackets[T packet.Packet](service *Service, addr netip.AddrPort, n int) <-chan T {
	return utils.MapChan[Signal, T](Intercept(service, func(sig Signal) bool {
		msg, ok := sig.(Message)
		if !ok { return false }

		_, ok = msg.Packet().(T)
		return ok && utils.Matches(addr, msg.Addr())
	}, n), func (sig Signal) T {
		return sig.(Message).Packet().(T)
	})
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptTCPPackets[T packet.Packet](service *Service, addr netip.AddrPort, n int) <-chan T {
	return utils.MapChan[Signal, T](Intercept(service, func(sig Signal) bool {
		msg, ok := sig.(TCPMessage)
		if !ok { return false }

		_, ok = msg.Packet().(T)
		return ok && utils.Matches(addr, msg.Addr())
	}, n), func (sig Signal) T {
		return sig.(TCPMessage).Packet().(T)
	})
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptUDPPackets[T packet.Packet](service *Service, addr netip.AddrPort, n int) <-chan T {
	return utils.MapChan[Signal, T](Intercept(service, func(sig Signal) bool {
		msg, ok := sig.(UDPMessage)
		if !ok { return false }

		_, ok = msg.Packet().(T)
		return ok && utils.Matches(addr, msg.Addr())
	}, n), func (sig Signal) T {
		return sig.(UDPMessage).Packet().(T)
	})
}



func InterceptTCPResponse[T packet.Packet](service *Service, request packet.Packet, addr netip.AddrPort) (T, error) {
	var aux <-chan T
	var err error
	
	service.PauseHandleWhile(func() {
		err = service.TCPServer().SendConnect(request, addr) //TODO: allow other sends?
		aux = InterceptTCPPackets[T](service, addr, 1)	//TODO: allow response from different port (set port to 0)
	})

	if err != nil {
		return *new(T), err
	} else {
		return <-aux, nil
	}
}

func InterceptUDPResponse[T packet.Packet](service *Service, request packet.Packet, addr netip.AddrPort) (T, error) {
	var aux <-chan T
	var err error
	
	service.PauseHandleWhile(func() {
		err = service.UDPServer().Send(request, addr)
		aux = InterceptUDPPackets[T](service, addr, 1)
	})

	if err != nil {
		return *new(T), err
	} else {
		return <-aux, nil
	}
}

//TODO: timeout