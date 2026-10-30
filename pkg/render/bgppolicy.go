package render

import (
	"fmt"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

type bgpPolicyStage int

const (
	bgpPolicyBefore bgpPolicyStage = iota
	bgpPolicyAfter
)

const (
	bgpPolicyIn  = "in"
	bgpPolicyOut = "out"

	bgpPolicyBeforeSeq = 100
	bgpPolicyUserSeq   = 1000
	bgpPolicyAfterSeq  = 2000
	bgpPolicyFinalSeq  = 9999
)

type bgpPolicyHook struct {
	Map       string
	Stage     bgpPolicyStage
	Direction string
	neighbors map[string]bool
}

func (h bgpPolicyHook) applies(addr, direction string) bool {
	return h.Direction == direction && h.neighbors[addr]
}

func bgpPolicyHooks(cfg *config.Config) []bgpPolicyHook {
	var hooks []bgpPolicyHook
	for _, p := range VRRPPrepends(cfg) {
		hooks = append(hooks, p.policyHook())
	}

	return hooks
}

func bgpPolicyHookLines(cfg *config.Config) []string {
	var lines []string
	for _, p := range VRRPPrepends(cfg) {
		lines = append(lines, p.routeMapLines()...)
	}

	return lines
}

func routeMapSafe(r rune) rune {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
		return r
	}

	return '_'
}

func bgpPolicyMapName(direction, addr, af string) string {
	return "ROUTIER-" + strings.ToUpper(direction) + "-" + strings.Map(routeMapSafe, addr) + "-" + af
}

func userPolicyMap(direction string, naf *config.BGPNeighborAF, family *config.BGPAddressFamily) string {
	if direction == bgpPolicyIn {
		if naf != nil && naf.RouteMapIn != "" {
			return naf.RouteMapIn
		}

		if family != nil {
			return family.RouteMapIn
		}

		return ""
	}

	if naf != nil && naf.RouteMapOut != "" {
		return naf.RouteMapOut
	}

	if family != nil {
		return family.RouteMapOut
	}

	return ""
}

func bgpPolicyMap(n config.BGPNeighbor, af, direction string) string {
	return bgpPolicyMapName(direction, n.Address, af)
}

func bgpPolicyRouteMaps(cfg *config.Config) string {
	if cfg == nil || cfg.Routing == nil || cfg.Routing.BGP == nil {
		return ""
	}

	bgp := cfg.Routing.BGP
	hooks := bgpPolicyHooks(cfg)
	lines := bgpPolicyHookLines(cfg)
	for _, af := range bgpAFs(bgp) {
		family := bgp.AddressFamilies[af]
		for _, n := range bgp.Neighbors {
			naf := bgpNeighborInAF(n, af)
			if naf == nil {
				continue
			}

			for _, direction := range []string{bgpPolicyIn, bgpPolicyOut} {
				name := bgpPolicyMapName(direction, n.Address, af)
				lines = append(lines, bgpPolicyWrapperLines(name, hooksFor(hooks, n.Address, direction), userPolicyMap(direction, naf, family))...)
			}
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return "\n" + strings.Join(lines, "\n")
}

func hooksFor(hooks []bgpPolicyHook, addr, direction string) []bgpPolicyHook {
	var out []bgpPolicyHook
	for _, h := range hooks {
		if h.applies(addr, direction) {
			out = append(out, h)
		}
	}

	return out
}

func callEntry(name string, seq int, target string) []string {
	return []string{fmt.Sprintf("route-map %s permit %d", name, seq), " call " + target, " on-match next", "exit"}
}

func bgpPolicyWrapperLines(name string, hooks []bgpPolicyHook, user string) []string {
	if user == "" {
		return []string{fmt.Sprintf("route-map %s deny %d", name, bgpPolicyFinalSeq), "exit"}
	}

	var lines []string
	before, after := bgpPolicyBeforeSeq, bgpPolicyAfterSeq
	for _, h := range hooks {
		if h.Stage == bgpPolicyBefore {
			lines = append(lines, callEntry(name, before, h.Map)...)
			before += 10
		}
	}

	lines = append(lines, callEntry(name, bgpPolicyUserSeq, user)...)
	for _, h := range hooks {
		if h.Stage == bgpPolicyAfter {
			lines = append(lines, callEntry(name, after, h.Map)...)
			after += 10
		}
	}

	return append(lines, fmt.Sprintf("route-map %s permit %d", name, bgpPolicyFinalSeq), "exit")
}
