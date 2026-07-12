package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestBGPNeighborExtraOptions(t *testing.T) {
	cfg := &config.Config{
		Hostname: "rtr",
		Routing: &config.Routing{
			BGP: &config.BGP{
				ASN: 65010,
				Neighbors: []config.BGPNeighbor{{
					Address: "198.51.100.1", RemoteASN: 65020,
					Passive: true, Shutdown: false,
					Extra: []string{"timers 10 30"},
					AddressFamilies: map[string]*config.BGPNeighborAF{
						"ipv4-unicast": {NextHopSelf: true, RouteReflectorClient: true, RemovePrivateAS: true, AllowASIn: 3, Weight: 100, Extra: []string{"send-community extended"}},
					},
				}},
			},
		},
	}
	out := renderFrr(t, cfg)
	for _, want := range []string{
		"neighbor 198.51.100.1 passive",
		"neighbor 198.51.100.1 timers 10 30",
		"neighbor 198.51.100.1 next-hop-self",
		"neighbor 198.51.100.1 route-reflector-client",
		"neighbor 198.51.100.1 remove-private-AS",
		"neighbor 198.51.100.1 allowas-in 3",
		"neighbor 198.51.100.1 weight 100",
		"neighbor 198.51.100.1 send-community extended",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}

	if strings.Contains(out, "shutdown") {
		t.Fatal("shutdown should not render when false")
	}
}
