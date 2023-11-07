package main

import (
	"encoding/json"
	"net/netip"
	"os"
	"slices"

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
	json.Unmarshal(bytes, &data)
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
				}
				
				var metrics utils.Metrics
				marshaled, err := json.Marshal(aux)
				if err != nil {
					panic(err.Error())
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
	return config
}