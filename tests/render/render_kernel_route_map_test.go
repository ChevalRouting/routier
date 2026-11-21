package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestFRRKernelRouteMap(t *testing.T) {
	cfg := &config.Config{
		Hostname: "rtr",
		Routing: &config.Routing{
			BGP: &config.BGP{
				ASN:      65000,
				RouterID: "10.0.0.1",
				RouteMaps: map[string][]config.RouteMapEntry{
					"LOOPBACK-SRC":  {{Seq: 10, Action: "permit", Set: map[string]string{"src": "10.0.0.1"}}},
					"LOOPBACK-SRC6": {{Seq: 10, Action: "permit", Set: map[string]string{"src": "2001:db8::1"}}},
				},
				AddressFamilies: map[string]*config.BGPAddressFamily{
					"ipv4-unicast": {KernelRouteMap: "LOOPBACK-SRC"},
					"ipv6-unicast": {KernelRouteMap: "LOOPBACK-SRC6"},
				},
			},
		},
	}

	out := renderFrr(t, cfg)
	for _, want := range []string{"ip protocol bgp route-map LOOPBACK-SRC\n", "ipv6 protocol bgp route-map LOOPBACK-SRC6\n", " set src 10.0.0.1\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in:\n%s", want, out)
		}
	}

	cfg.Routing.BGP.AddressFamilies = map[string]*config.BGPAddressFamily{"ipv4-unicast": {}}
	if out := renderFrr(t, cfg); strings.Contains(out, "protocol bgp route-map") {
		t.Fatalf("rendered a kernel route-map without one configured:\n%s", out)
	}
}
