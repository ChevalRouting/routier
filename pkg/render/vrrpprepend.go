package render

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

const VRRPPrependMapPrefix = "ROUTIER-VRRP-"

type VRRPPrepend struct {
	Instance  string
	Map       string
	Set       string
	Seqs      []int
	neighbors map[string]bool
	prefixes4 []string
	prefixes6 []string
}

type prefixFamily struct {
	list   string
	match  string
	maxLen int
}

var (
	prefixFamily4 = prefixFamily{list: "ip prefix-list", match: "match ip address prefix-list", maxLen: 32}
	prefixFamily6 = prefixFamily{list: "ipv6 prefix-list", match: "match ipv6 address prefix-list", maxLen: 128}
)

func VRRPPrepends(cfg *config.Config) []VRRPPrepend {
	if cfg == nil || cfg.HA == nil || cfg.Routing == nil || cfg.Routing.BGP == nil {
		return nil
	}

	bgp := cfg.Routing.BGP

	var out []VRRPPrepend
	for _, v := range cfg.HA.VRRP {
		if v.BGPPrepend == nil {
			continue
		}

		p := VRRPPrepend{
			Instance:  v.KeepalivedName(),
			Map:       VRRPPrependMapPrefix + strconv.Itoa(v.ID),
			Set:       prependSet(bgp.ASN, v.BGPPrepend.CountOrDefault()),
			neighbors: prependNeighbors(bgp, v.BGPPrepend),
		}
		p.prefixes4, p.prefixes6 = splitPrefixes(v.BGPPrepend.Prefixes)
		p.Seqs = prependSeqs(p)
		out = append(out, p)
	}

	return out
}

func splitPrefixes(prefixes []string) ([]string, []string) {
	var v4, v6 []string
	for _, prefix := range prefixes {
		_, network, err := net.ParseCIDR(prefix)
		if err != nil {
			continue
		}

		if network.IP.To4() != nil {
			v4 = append(v4, network.String())
		} else {
			v6 = append(v6, network.String())
		}
	}

	return v4, v6
}

func prependSeqs(p VRRPPrepend) []int {
	if len(p.prefixes4) == 0 && len(p.prefixes6) == 0 {
		return []int{10}
	}

	var seqs []int
	if len(p.prefixes4) > 0 {
		seqs = append(seqs, 10)
	}

	if len(p.prefixes6) > 0 {
		seqs = append(seqs, 20)
	}

	return seqs
}

func (p VRRPPrepend) scoped() bool {
	return len(p.prefixes4) > 0 || len(p.prefixes6) > 0
}

func prefixListLines(name string, fam prefixFamily, prefixes []string) []string {
	lines := make([]string, 0, len(prefixes))
	for i, prefix := range prefixes {
		line := fmt.Sprintf("%s %s seq %d permit %s", fam.list, name, (i+1)*5, prefix)
		if prefixLen(prefix) < fam.maxLen {
			line += fmt.Sprintf(" le %d", fam.maxLen)
		}

		lines = append(lines, line)
	}

	return lines
}

func prefixLen(prefix string) int {
	_, network, err := net.ParseCIDR(prefix)
	if err != nil {
		return 0
	}

	ones, _ := network.Mask.Size()
	return ones
}

func (p VRRPPrepend) routeMapLines() []string {
	if !p.scoped() {
		return []string{"route-map " + p.Map + " permit 10", " set " + p.Set, "exit"}
	}

	var lines []string
	if len(p.prefixes4) > 0 {
		name := p.Map + "-V4"
		lines = append(lines, prefixListLines(name, prefixFamily4, p.prefixes4)...)
		lines = append(lines, "route-map "+p.Map+" permit 10", " "+prefixFamily4.match+" "+name, " set "+p.Set, "exit")
	}

	if len(p.prefixes6) > 0 {
		name := p.Map + "-V6"
		lines = append(lines, prefixListLines(name, prefixFamily6, p.prefixes6)...)
		lines = append(lines, "route-map "+p.Map+" permit 20", " "+prefixFamily6.match+" "+name, " set "+p.Set, "exit")
	}

	return append(lines, "route-map "+p.Map+" permit 100", "exit")
}

func prependSet(asn, count int) string {
	asns := make([]string, count)
	for i := range asns {
		asns[i] = strconv.Itoa(asn)
	}

	return "as-path prepend " + strings.Join(asns, " ")
}

func prependNeighbors(bgp *config.BGP, p *config.VRRPBGPPrepend) map[string]bool {
	out := map[string]bool{}
	if len(p.Neighbors) > 0 {
		for _, addr := range p.Neighbors {
			out[addr] = true
		}

		return out
	}

	for _, n := range bgp.Neighbors {
		if n.RemoteASN != bgp.ASN {
			out[n.Address] = true
		}
	}

	return out
}

func (p VRRPPrepend) policyHook() bgpPolicyHook {
	return bgpPolicyHook{
		Map:       p.Map,
		Stage:     bgpPolicyBefore,
		Direction: bgpPolicyOut,
		neighbors: p.neighbors,
	}
}
