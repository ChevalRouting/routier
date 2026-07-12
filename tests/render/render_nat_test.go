package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/nat"
)

func TestNatRulesRenderIntoChains(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "rtr",
		Interfaces: map[string]*config.Interface{"wan": {Device: "eth0", Addresses: []string{"203.0.113.2/24"}}},
	}

	nat.Apply(cfg, []nat.Spec{
		{Kind: nat.KindMasquerade, Out: "$wan_interfaces"},
		{Kind: nat.KindDNAT, In: "$wan_interfaces", Proto: "tcp", DPort: "443", To: "10.0.0.5:8443"},
	})

	out := renderNft(t, cfg)

	for _, want := range []string{
		"chain postrouting {",
		"chain prerouting {",
		`oifname $wan_interfaces masquerade comment "routier:nat"`,
		`iifname $wan_interfaces tcp dport 443 dnat ip to 10.0.0.5:8443 comment "routier:nat"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in rendered ruleset:\n%s", want, out)
		}
	}
}
