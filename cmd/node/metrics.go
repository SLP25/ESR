package main

import (
	"bytes"
	"log/slog"
	"net/netip"
	"time"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/service"
	"github.com/SLP25/ESR/internal/utils"
)

//The timeout is the same as the interval between packets
//The bandwidth isn't calculated, and is returned as 0
func measureMetrics(address netip.AddrPort, packets int, interval time.Duration) (utils.Metrics, error) {
	var port uint16
	var server service.UDPServer

	err := server.Open(&port)
	if err != nil { return utils.Metrics{}, err}
	defer server.Close()

	var totalLatency time.Duration
	receivedPackets := 0

	for i := 0; i < packets; i++ {
		p := packet.NewPing()
		err := server.Send(p, address)
		if err != nil { return utils.Metrics{}, err }

		sent := time.Now()
		timeout := time.After(interval)
		L: for {
			select {
				case <-timeout: break L
				case msg := <-server.Output():
					resp, err := packet.Deserialize(bytes.NewReader(msg.Data))
					if err != nil { slog.Warn("Error deserializing response to ping", "err", err); continue L }
					
					ping, ok := resp.(packet.Ping)
					if !ok { slog.Warn("Received invalid response to ping", "response", resp); continue L }
					
					if ping.ID != p.ID { slog.Warn("Received ping with invalid ID", "sendID", p.ID, "receivedID", ping.ID); continue L }
			
					totalLatency += time.Now().Sub(sent)
					receivedPackets++
			}
		}
	}

	var latency time.Duration
	if receivedPackets == 0 {
		latency = time.Hour
	} else {
		latency = totalLatency / time.Duration(receivedPackets)
	}

	metrics := utils.Metrics{
		Latency: latency,
		PacketLoss: float64(packets - receivedPackets) / float64(packets),
	}

	return metrics, nil
}


type metricsMonitor struct {
	metrics map[netip.AddrPort]utils.Metrics
	cancel chan<- struct{}
}

func (this metricsMonitor) updateMetrics(addr netip.AddrPort) {
    m, err := measureMetrics(addr, 10, 200 * time.Millisecond)
    if err != nil {
        slog.Error("Error updating metrics", "addr", addr, "err", err)
        return
    }

    this.metrics[addr] = m
	slog.Debug("Calculated new metrics for server", "addr", addr, "metrics", m)
}

func (this *node) monitorMetrics(servers []netip.AddrPort) metricsMonitor {
	cancel := make(chan struct{})
	ans := metricsMonitor{
		metrics: make(map[netip.AddrPort]utils.Metrics),
		cancel: cancel,
	}

	go func() {
		for {
			for _, s := range servers {
				go ans.updateMetrics(s)
			}
	
			select {
				case <-cancel: return
				case <-time.After(10 * time.Second):
			}
		}
	}()

	return ans
}

func (this metricsMonitor) GetMetrics(server netip.AddrPort) utils.Metrics {
	return this.metrics[server]
}

func (this metricsMonitor) Stop() {
	this.cancel <- struct{}{}
	close(this.cancel)
}