package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func TestNftVRRPVipDefines(t *testing.T) {
	cfg := &config.Config{
		Hostname: "rtr",
		Interfaces: map[string]*config.Interface{
			"wan": {
				Device:    "eth0",
				Addresses: []string{"203.0.113.2/24"},
			},
		},
		HA: &config.HA{
			VRRP: []*config.VRRPInstance{
				{ID: 10, Interface: "wan", VIPs: []string{"203.0.113.1/24", "198.51.100.1/24"}},
			},
		},
	}

	byName := map[string]string{}
	for _, v := range render.NftVars(cfg) {
		byName[v.Name] = v.Value
	}

	for _, name := range []string{
		"wan_vrrp_address", "wan_vrrp_network", "wan_vrrp_addresses",
		"vrrp_10_address", "vrrp_10_addresses",
	} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("missing vrrp define %q; got: %+v", name, byName)
		}
	}

	if !strings.Contains(byName["wan_vrrp_addresses"], "203.0.113.1") || !strings.Contains(byName["wan_vrrp_addresses"], "198.51.100.1") {
		t.Fatalf("wan_vrrp_addresses should combine every VIP, got: %q", byName["wan_vrrp_addresses"])
	}
}
