package service

import (
	"net/netip"

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

		this.ans <- sig

		if this.n == 0 {
			close(this.ans)
		}

		return true
	}

	return false
}

// Intercepts the first n signals that satisfy the condition
// If n is non-positive, all signals that satisfy the condition are intercepted
func Intercept(service *Service, accept checker, n int) chan Signal {
	i := new(interceptor)
	*i = interceptor{service: service, ans: make(chan Signal), accept: accept, n: n}
	service.AddHandler(i)
	return i.ans
}

// Intercepts the first n messages from the specified remote address
// If n is non-positive, all messages from the remote address are intercepted
func InterceptMessages(service *Service, addr netip.AddrPort, n int) chan Message {
	return utils.CastChan[Signal, Message](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case Message:
			return sig.(Message).Addr() == addr
		default:
			return false
		}
	}, n))
}

// Intercepts the first n signals of the specified type
// If n is non-positive, all signals of the specified type are intercepted
func InterceptSignal[T Signal](service *Service, n int) chan T {
	return utils.CastChan[Signal, T](Intercept(service, func(sig Signal) bool {
		switch sig.(type) {
		case T:
			return true
		default:
			return false
		}
	}, n))
}

//TODO: InterceptTCP, IntersectUDP