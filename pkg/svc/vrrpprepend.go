package svc

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/rs/zerolog/log"
)

const vtyshPrependTimeout = 15 * time.Second

type routeMapEntry struct {
	Map string
	Seq int
}

func vrrpPrependArgs(p render.VRRPPrepend, seqs []int, master bool) []string {
	set := "set " + p.Set
	if master {
		set = "no set as-path prepend"
	}

	args := []string{"vtysh", "-c", "configure terminal"}
	for _, seq := range seqs {
		args = append(args, "-c", fmt.Sprintf("route-map %s permit %d", p.Map, seq), "-c", set)
	}

	return args
}

func runningPrepends() (map[routeMapEntry]string, error) {
	out, err := runCombined([]string{"vtysh", "-c", "show running-config bgpd"}, vtyshPrependTimeout)
	if err != nil {
		return nil, fmt.Errorf("read bgpd running config: %w", err)
	}

	return parseRunningPrepends(out), nil
}

func parseRunningPrepends(running []byte) map[routeMapEntry]string {
	entries := map[routeMapEntry]string{}
	var current *routeMapEntry

	scanner := bufio.NewScanner(bytes.NewReader(running))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		switch {
		case len(fields) == 4 && fields[0] == "route-map" && fields[2] == "permit":
			var seq int
			if _, err := fmt.Sscanf(fields[3], "%d", &seq); err != nil {
				current = nil
				continue
			}

			current = &routeMapEntry{Map: fields[1], Seq: seq}
			entries[*current] = ""
		case current == nil:
		case !strings.HasPrefix(line, " "):
			current = nil
		case len(fields) > 2 && fields[0] == "set" && fields[1] == "as-path" && fields[2] == "prepend":
			entries[*current] = strings.Join(fields[1:], " ")
		}
	}

	return entries
}

func stalePrependSeqs(p render.VRRPPrepend, running map[routeMapEntry]string, master bool) ([]int, error) {
	want := p.Set
	if master {
		want = ""
	}

	var stale []int
	for _, seq := range p.Seqs {
		have, ok := running[routeMapEntry{Map: p.Map, Seq: seq}]
		if !ok {
			return nil, fmt.Errorf("route-map %s permit %d is not loaded in bgpd", p.Map, seq)
		}

		if have != want {
			stale = append(stale, seq)
		}
	}

	return stale, nil
}

func bgpPrependHook(cfg *config.Config, t VRRPTransition) error {
	var prepends []render.VRRPPrepend
	for _, p := range render.VRRPPrepends(cfg) {
		if p.Instance == t.Instance.KeepalivedName() {
			prepends = append(prepends, p)
		}
	}

	if len(prepends) == 0 {
		return nil
	}

	running, err := runningPrepends()
	if err != nil {
		return err
	}

	for _, p := range prepends {
		stale, err := stalePrependSeqs(p, running, t.Master())
		if err != nil {
			return err
		}

		if len(stale) == 0 {
			continue
		}

		if err := runCmdOutput(vrrpPrependArgs(p, stale, t.Master()), vtyshPrependTimeout); err != nil {
			return err
		}

		log.Info().Str("instance", p.Instance).Str("state", t.State).Str("route_map", p.Map).Ints("seqs", stale).Bool("prepend", !t.Master()).Msg("vrrp: synced bgp prepend")
	}

	return nil
}
