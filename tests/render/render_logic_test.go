package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func logicConfig() *config.Config {
	return &config.Config{
		Hostname: "rtr",
		VRFs:     map[string]*config.VRFConfig{"blue": {Table: 10}},
		Interfaces: map[string]*config.Interface{
			"wan":  {Device: "eth0", Addresses: []string{"192.0.2.2/24", "2001:db8::2/64"}},
			"core": {Device: "eth1", Addresses: []string{"10.0.0.1/24", "2001:db8:1::1/64"}},
			"cust": {Device: "eth2", VRF: "blue", Addresses: []string{"10.9.0.1/24", "2001:db8:9::1/64"}},
			"unn":  {Device: "eth3", Addresses: []string{"10.255.0.1/32"}},
		},
		Routing: &config.Routing{
			Static: []config.StaticRoute{
				{Destination: "0.0.0.0/0", Via: "192.0.2.1", Dev: "eth0", Metric: 100},
				{Destination: "0.0.0.0/0", Via: "192.0.2.254", Dev: "wan", Metric: 200},
			},
			BGP: &config.BGP{
				ASN: 65000, RouterID: "10.0.0.1",
				AddressFamilies: map[string]*config.BGPAddressFamily{"ipv4-unicast": {DefaultOriginate: true}},
				Neighbors: []config.BGPNeighbor{
					{Address: "192.0.2.1", RemoteASN: 64500, Description: "transit"},
					{Address: "192.0.2.254", RemoteASN: 64501, Description: "transit"},
				},
			},
			OSPF: &config.OSPF{
				RouterID: "10.0.0.1", AllowInbound: []string{"unn"},
				Interfaces: map[string]*config.OSPFInterface{"unn": {Area: "0"}},
			},
			OSPF6: &config.OSPF6{
				RouterID: "10.0.0.1", AllowInbound: []string{"core"}, PassiveInterfaces: []string{"wan"},
				Areas:      []config.OSPF6Area{{ID: "0", DefaultCost: 10}},
				Interfaces: map[string]*config.OSPF6Interface{"core": {Area: "0"}},
			},
			VRFs: map[string]*config.VRFRouting{"blue": {
				BGP: &config.BGP{
					ASN: 65000, RouterID: "10.9.0.1", NoImportCheck: true,
					Neighbors:   []config.BGPNeighbor{{Address: "10.9.0.2", RemoteASN: 64600}},
					PrefixLists: map[string][]config.PrefixEntry{"BLUE-IN": {{Seq: 10, Action: "permit", Prefix: "10.0.0.0/8", LE: 24}}},
				},
				OSPF6: &config.OSPF6{
					RouterID: "10.9.0.1", AllowInbound: []string{"cust"},
					Interfaces: map[string]*config.OSPF6Interface{"cust": {Area: "0"}},
				},
			}},
		},
		HA: &config.HA{VRRP: []*config.VRRPInstance{{ID: 1, Interface: "core", VIPs: []string{"10.0.0.254/24", "2001:db8:1::fe/64"}}}},
		Nftables: &config.NftablesConfig{Chains: map[string]*config.NftChain{
			"postrouting": {Managed: []config.ManagedRule{
				{Action: "masquerade", Match: &config.RuleMatch{OIF: "$wan_interfaces", SAddr: "$core_network6"}},
				{Action: "masquerade", Match: &config.RuleMatch{OIF: "$wan_interfaces", AddrFamily: "ip"}},
			}},
		}},
	}
}

func renderOutputByName(t *testing.T, cfg *config.Config, name string) string {
	t.Helper()

	outs, err := render.All(cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, o := range outs {
		if o.Name == name {
			return o.Content
		}
	}

	t.Fatalf("no %s output", name)

	return ""
}

func assertContains(t *testing.T, out string, want ...string) {
	t.Helper()

	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

func assertMissing(t *testing.T, out string, unwanted ...string) {
	t.Helper()

	for _, u := range unwanted {
		if strings.Contains(out, u) {
			t.Errorf("unexpected %q in:\n%s", u, out)
		}
	}
}

func TestFRRRendersCommandsFRRAccepts(t *testing.T) {
	out := renderFrr(t, logicConfig())

	assertContains(t, out,
		"router bgp 65000 vrf blue\n bgp router-id 10.9.0.1\n bgp log-neighbor-changes\n no bgp network import-check",
		"  neighbor 192.0.2.1 default-originate",
		"  neighbor 192.0.2.254 default-originate",
		"interface eth0\n ipv6 ospf6 passive",
		"interface eth2\n ipv6 ospf6 area 0",
		"ip prefix-list BLUE-IN seq 10 permit 10.0.0.0/8 le 24",
	)
	assertMissing(t, out, "no bgp import-check", "default-information originate", "passive-interface wan", "default-cost")
}

func TestNftDefinesStayUnique(t *testing.T) {
	out := renderNft(t, logicConfig())

	assertContains(t, out,
		"define wan_gateway = { 192.0.2.1, 192.0.2.254 }",
		"define bgp_neighbor_transit = { 192.0.2.1, 192.0.2.254 }",
	)

	seen := map[string]bool{}
	for line := range strings.Lines(out) {
		if rest, ok := strings.CutPrefix(line, "define "); ok {
			name, _, _ := strings.Cut(rest, " ")
			if seen[name] {
				t.Errorf("define %s emitted twice", name)
			}

			seen[name] = true
		}
	}
}

func TestNftRoutingAllowRules(t *testing.T) {
	out := renderNft(t, logicConfig())

	assertContains(t, out,
		`iifname $core_interfaces ip6 saddr fe80::/10 meta l4proto 89 accept comment "routier: ospf6"`,
		`iifname $cust_interfaces ip6 saddr fe80::/10 meta l4proto 89 accept comment "routier: ospf6"`,
		`iifname $unn_interfaces meta l4proto 89 accept comment "routier: ospf"`,
	)
}

func TestNftManagedRuleFamilies(t *testing.T) {
	out := renderNft(t, logicConfig())

	assertContains(t, out,
		"oifname $wan_interfaces ip6 saddr $core_network6 masquerade",
		"oifname $wan_interfaces meta nfproto ipv4 masquerade",
	)
}

func TestKeepalivedExcludesOtherFamilyVIPs(t *testing.T) {
	out := renderOutputByName(t, logicConfig(), "keepalived/keepalived.conf")

	assertContains(t, out,
		"virtual_ipaddress {\n        10.0.0.254/24\n    }",
		"virtual_ipaddress_excluded {\n        2001:db8:1::fe/64\n    }",
	)
}
