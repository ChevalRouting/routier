package nat

import (
	"reflect"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestApplyParseRoundTrip(t *testing.T) {
	specs := []Spec{
		{Kind: KindMasquerade, Out: "$wan_interfaces", Source: "$lan_network", Family: "ip"},
		{Kind: KindSNAT, Out: "$wan_interfaces", To: "203.0.113.5"},
		{Kind: KindDNAT, In: "$wan_interfaces", Proto: "tcp", DPort: "443", To: "10.0.0.5:8443"},
	}

	cfg := &config.Config{}
	Apply(cfg, specs)

	got := Parse(cfg.Nftables)
	if !reflect.DeepEqual(got, specs) {
		t.Fatalf("round-trip mismatch:\n want %+v\n got  %+v", specs, got)
	}
}

func TestApplyPreservesUserRulesAndReplacesTagged(t *testing.T) {
	userRule := config.ManagedRule{Action: "accept", Match: &config.RuleMatch{DPort: "22"}}
	staleNat := config.ManagedRule{Tag: Tag, Action: "masquerade", Match: &config.RuleMatch{OIF: "$old"}}

	cfg := &config.Config{Nftables: &config.NftablesConfig{
		Chains: map[string]*config.NftChain{
			"postrouting": {Managed: []config.ManagedRule{userRule, staleNat}},
		},
	}}

	Apply(cfg, []Spec{{Kind: KindMasquerade, Out: "$wan_interfaces"}})

	got := cfg.Nftables.Chains["postrouting"].Managed
	if len(got) != 2 {
		t.Fatalf("expected user rule + 1 new nat rule, got %d: %+v", len(got), got)
	}

	if got[0].Action != "accept" || got[0].Tag != "" {
		t.Fatalf("user rule not preserved as first entry: %+v", got[0])
	}

	if got[1].Tag != Tag || got[1].Match.OIF != "$wan_interfaces" {
		t.Fatalf("stale nat rule not replaced by new one: %+v", got[1])
	}
}

func TestApplyEmptyPrunesChainItCreated(t *testing.T) {
	cfg := &config.Config{}
	Apply(cfg, []Spec{{Kind: KindMasquerade, Out: "$wan_interfaces"}})
	Apply(cfg, nil)

	if _, ok := cfg.Nftables.Chains["postrouting"]; ok {
		t.Fatalf("expected empty postrouting chain to be pruned, chains: %+v", cfg.Nftables.Chains)
	}
}
