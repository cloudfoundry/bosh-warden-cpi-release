package vm

import (
	"encoding/json"
	"fmt"
	"net"
)

type Networks map[string]Network

type Network struct {
	Type string

	IP      string
	Netmask string
	Gateway string

	DNS     []string
	Default []string

	CloudProperties map[string]interface{}
}

func (ns Networks) Default() Network {
	var n Network

	for _, n = range ns {
		if n.IsDefaultFor("gateway") {
			break
		}
	}

	return n // returns last network
}

func (ns Networks) BackfillDefaultDNS(nameservers []string) Networks {
	bytes, err := json.Marshal(ns)
	if err != nil {
		panic("struct marshaling failed")
	}

	var backfilledNs Networks

	err = json.Unmarshal(bytes, &backfilledNs)
	if err != nil {
		panic("struct marshaling failed")
	}

	for name, n := range backfilledNs {
		if n.IsDefaultFor("dns") {
			if len(n.DNS) == 0 {
				n.DNS = nameservers
				backfilledNs[name] = n
			}
			break
		}
	}

	return backfilledNs
}

func (n Network) IsDefaultFor(what string) bool {
	for _, def := range n.Default {
		if def == what {
			return true
		}
	}

	return false
}

func (n Network) IsDynamic() bool { return n.Type == "dynamic" }

func (n Network) IPWithSubnetMask() string {
	ones, _ := net.IPMask(net.ParseIP(n.Netmask).To4()).Size()
	return fmt.Sprintf("%s/%d", n.IP, ones)
}
