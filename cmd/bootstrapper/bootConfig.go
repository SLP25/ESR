package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"slices"

	"github.com/SLP25/ESR/internal/packet"
	"github.com/SLP25/ESR/internal/utils"
)

type pair struct {
	first string
	second string
}

type config struct {
	servers []netip.AddrPort
	nodes map[string] netip.AddrPort
	edges map[pair] utils.Metrics 
	rp string
}

func readField(dict map[string]any, field string) any {
	if val, ok := dict[field]; ok {
		return val
	} else {
		panic("No field '" + field + "' in boot config")
	}
}

func MustReadConfig(filename string) config {
	bytes, err := os.ReadFile(filename)
	if err != nil { panic(err.Error()) }

	var data map[string]any
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		panic("Error parsing boot config: " + err.Error())
	}

	config := config{nodes: make(map[string]netip.AddrPort), edges: make(map[pair]utils.Metrics)}

	for _, aux := range readField(data, "servers").([]any) {
		addr := netip.MustParseAddrPort(aux.(string))
		
		if slices.ContainsFunc(config.servers, func (a netip.AddrPort) bool {
			return addr.Addr() == a.Addr() && addr.Port() == a.Port()
		}) {
			panic("Repeated server IP in boot config")
		}

		config.servers = append(config.servers, addr)
	}

	for k, v := range readField(data, "nodes").(map[string]any) {
		if utils.ContainsKey(config.nodes, k) {
			panic("Repeated node name in boot config")
		}

		config.nodes[k] = netip.MustParseAddrPort(v.(string))
	}

	for _, e := range readField(data, "edges").([]any) {
		aux := e.(map[string]any)
		done := false

		for k, v := range aux {
			if utils.ContainsKey(config.nodes, k) && utils.ContainsKey(config.nodes, v.(string)) {
				edge := pair{first: k,second: v.(string)}
				delete(aux, k)
				
				if utils.ContainsKey[pair](config.edges, edge) {
					panic("Repeated edge in boot config")
				} else if edge.first == edge.second {
					panic("Self-loop in boot config not allowed")
				}
				
				var metrics utils.Metrics
				marshaled, err := json.Marshal(aux)
				if err != nil {
					panic(fmt.Sprintf("Error deserializing metrics in node %s<->%s: %s", edge.first, edge.second, err.Error()))
				} else if metrics.Bandwidth == 0 {
					panic(fmt.Sprintf("Invalid value for Bandwidth: 0. Did you forget to specify the Bandwidth for edge %s<->%s?", edge.first, edge.second))
				}

				json.Unmarshal(marshaled, &metrics)
				config.edges[edge] = metrics
				done = true
			}
		}

		if !done {
			panic("Invalid edge in boot config")
		}
	}

	config.rp = readField(data, "rp").(string)
	if !utils.ContainsKey(config.nodes, config.rp) {
		panic("RP not registered as a node in boot config: " + config.rp)
	}

	return config
}

func (this *config) getName(node netip.Addr) (string, error) {
	for name, n := range this.nodes {
		if n.Addr() == node {
			return name, nil
		}
	}

	return "", errors.New(node.String() + " not in boot config")
} 

func (this *config) getNeighbours(node netip.Addr) (map[netip.AddrPort]utils.Metrics, error) {
	n, err := this.getName(node)
	neighbours := make(map[netip.AddrPort]utils.Metrics)

	if err != nil {
		return neighbours, err
	}


	for edge, metrics := range this.edges {
		if edge.first == n {
			neighbours[this.nodes[edge.second]] = metrics
		} else if edge.second == n {
			neighbours[this.nodes[edge.first]] = metrics
		}
	}

	return neighbours, nil
}

func (this *config) BootNode(node netip.Addr) (packet.StartupResponseNode, error) {
	neighbours, err := this.getNeighbours(node)
	if err != nil {
		return packet.StartupResponseNode{}, err
	}

	var servers []netip.AddrPort
	if node == this.nodes[this.rp].Addr() {
		servers = this.servers
	}

	return packet.StartupResponseNode{Neighbours: neighbours, Servers: servers}, nil
}