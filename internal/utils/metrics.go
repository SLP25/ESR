package utils

type Metrics struct {
	Latency int	//in ms
	//Throughput int
}

func (this Metrics) Compose(m Metrics) Metrics {
	return Metrics{Latency: this.Latency + m.Latency/*, Throughput: min(this.Throughput, m.Throughput)*/}
}

func (this Metrics) BetterThan(m Metrics) bool {
	return this.Latency <= m.Latency
}


type StreamMetadata struct {
	Throughput int
}