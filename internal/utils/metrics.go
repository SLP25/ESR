package utils

import "time"

type Metrics struct {
	Latency time.Duration	//in ms
	PacketLoss float64 //from 0 to 1
	Bandwidth int
}

func (this Metrics) Compose(m Metrics) Metrics {
	return Metrics{
		Latency: this.Latency + m.Latency,
		PacketLoss: 1 - (1 - this.PacketLoss) * (1 - m.PacketLoss),
		/*, Bandwith: min(this.Bandwith, m.Bandwith)*/}
}

func (this Metrics) BetterThan(m Metrics) bool {
	return float64(this.Latency.Milliseconds()) + this.PacketLoss * 5000 <= float64(m.Latency.Milliseconds()) + m.PacketLoss * 5000
}


type StreamMetadata struct {
	Bitrate int
}