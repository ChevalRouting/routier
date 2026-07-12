package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func renderFrr(t *testing.T, cfg *config.Config) string {
	t.Helper()

	outs, err := render.All(cfg)
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	for _, o := range outs {
		if o.Name == "frr/frr.conf" {
			return o.Content
		}
	}

	t.Fatal("frr/frr.conf not rendered")
	return ""
}

func TestBGPNeighborAFDisableActivate(t *testing.T) {
	mkCfg := func(disabled bool) *config.Config {
		return &config.Config{
			Hostname: "rtr",
			Routing: &config.Routing{
				BGP: &config.BGP{
					ASN: 65010,
					Neighbors: []config.BGPNeighbor{
						{
							Address:   "198.51.100.1",
							RemoteASN: 65020,
							AddressFamilies: map[string]*config.BGPNeighborAF{
								"ipv4-unicast": {Disabled: disabled},
							},
						},
					},
				},
			},
		}
	}

	enabled := renderFrr(t, mkCfg(false))
	if !strings.Contains(enabled, "neighbor 198.51.100.1 activate") || strings.Contains(enabled, "no neighbor 198.51.100.1 activate") {
		t.Fatalf("enabled AF should emit activate, got:\n%s", enabled)
	}

	disabled := renderFrr(t, mkCfg(true))
	if !strings.Contains(disabled, "no neighbor 198.51.100.1 activate") {
		t.Fatalf("disabled AF should emit `no ... activate`, got:\n%s", disabled)
	}
}
