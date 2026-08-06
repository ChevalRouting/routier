package rendertest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func renderNft(t *testing.T, cfg *config.Config) string {
	t.Helper()
	outs, err := render.All(cfg)
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	for _, o := range outs {
		if o.Name == "nftables/routier.nft" {
			return o.Content
		}
	}

	t.Fatal("nftables/routier.nft not rendered")
	return ""
}

func TestNftRoutierOwnsChains(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "rtr",
		Interfaces: map[string]*config.Interface{"wan": {Device: "eth0", Addresses: []string{"203.0.113.2/24"}}},
		Nftables: &config.NftablesConfig{
			Chains: map[string]*config.NftChain{
				"input": {
					Policy: "drop",
					Rules:  "tcp dport 443 accept",
					Managed: []config.ManagedRule{
						{Match: &config.RuleMatch{Protocol: "tcp", DPort: "179"}, Action: "accept", Comment: "bgp"},
					},
				},
				"postrouting": {
					Rules: `oifname "eth0" masquerade`,
				},
			},
		},
	}

	out := renderNft(t, cfg)

	if !strings.Contains(out, "table inet routier {") {
		t.Fatalf("expected routier table, got:\n%s", out)
	}

	for _, want := range []string{
		"type filter hook input priority filter; policy drop;",
		"type filter hook forward priority filter; policy drop;",
		"type filter hook output priority filter; policy accept;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q, got:\n%s", want, out)
		}
	}

	if strings.Contains(out, "tcp dport { ssh, 8080 } accept") {
		t.Fatalf("base rules should not be injected, got:\n%s", out)
	}

	if !strings.Contains(out, "tcp dport 443 accept") {
		t.Fatalf("expected user rule, got:\n%s", out)
	}

	if !strings.Contains(out, `tcp dport 179 accept comment "bgp"`) {
		t.Fatalf("expected managed rule, got:\n%s", out)
	}

	if !strings.Contains(out, "type nat hook postrouting priority srcnat; policy accept;") ||
		!strings.Contains(out, `oifname "eth0" masquerade`) {
		t.Fatalf("expected postrouting nat chain, got:\n%s", out)
	}

	if strings.Contains(out, "hook prerouting") {
		t.Fatalf("prerouting should not be emitted when unused, got:\n%s", out)
	}
}

func TestNftDefinesAndPolicyOverride(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "rtr",
		Interfaces: map[string]*config.Interface{"wan": {Device: "eth0", Addresses: []string{"203.0.113.2/24"}}},
		Nftables: &config.NftablesConfig{
			Defines: "define my_set = { 10.0.0.0/8 }",
			Chains: map[string]*config.NftChain{
				"input": {Policy: "accept", Rules: "ip saddr $my_set accept"},
			},
		},
	}
	out := renderNft(t, cfg)

	defIdx := strings.Index(out, "define my_set =")
	tableIdx := strings.Index(out, "table inet routier {")
	if defIdx < 0 || tableIdx < 0 || defIdx > tableIdx {
		t.Fatalf("custom define must precede routier table, got:\n%s", out)
	}

	if !strings.Contains(out, "type filter hook input priority filter; policy accept;") {
		t.Fatalf("expected input policy override to accept, got:\n%s", out)
	}
}

func TestNftFileAndInclude(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "in.nft"), []byte("ip daddr 10.0.0.0/8 accept\n\nmeta l4proto icmp accept\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "custom.nft"), []byte("table ip my_nat {\n  chain x {}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Hostname:   "rtr",
		BaseDir:    dir,
		Interfaces: map[string]*config.Interface{"wan": {Device: "eth0", Addresses: []string{"203.0.113.2/24"}}},
		Nftables: &config.NftablesConfig{
			Chains:  map[string]*config.NftChain{"forward": {Files: []string{"in.nft"}}},
			Include: []string{"custom.nft"},
		},
	}
	out := renderNft(t, cfg)

	if !strings.Contains(out, "ip daddr 10.0.0.0/8 accept") {
		t.Fatalf("expected file rule injected into forward chain, got:\n%s", out)
	}

	tableIdx := strings.Index(out, "table inet routier {")
	incIdx := strings.Index(out, "table ip my_nat {")
	if incIdx < 0 || incIdx < tableIdx {
		t.Fatalf("expected include rendered after routier table, got:\n%s", out)
	}
}

func TestNftVRFDefines(t *testing.T) {
	cfg := &config.Config{
		Hostname: "rtr",
		VRFs: map[string]*config.VRFConfig{
			"vm-public": {Table: 100},
			"mgmt":      {Table: 200},
		},
		Interfaces: map[string]*config.Interface{
			"wan":  {Device: "eth0", Addresses: []string{"203.0.113.2/24"}, VRF: "vm-public"},
			"wan2": {Device: "eth1", VRF: "vm-public"},
		},
	}
	out := renderNft(t, cfg)

	if !strings.Contains(out, `define vrf_vm_public_interfaces = "vm-public"`) {
		t.Fatalf("expected vrf master device define, got:\n%s", out)
	}

	if !strings.Contains(out, `define vrf_vm_public_members = { "eth0", "eth1" }`) {
		t.Fatalf("expected vrf members define, got:\n%s", out)
	}

	if !strings.Contains(out, `define vrf_mgmt_interfaces = "mgmt"`) {
		t.Fatalf("expected second vrf interface define, got:\n%s", out)
	}

	if strings.Contains(out, "define vrf_mgmt_members") {
		t.Fatalf("vrf with no members should not emit a members define, got:\n%s", out)
	}

	if !strings.Contains(out, `define vrfs = { "mgmt", "vm-public" }`) {
		t.Fatalf("expected vrfs aggregate define, got:\n%s", out)
	}
}

func TestNftVRFNameCollision(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "rtr",
		VRFs:       map[string]*config.VRFConfig{"wan": {Table: 100}},
		Interfaces: map[string]*config.Interface{"wan": {Device: "eth0", VRF: "wan"}},
	}
	out := renderNft(t, cfg)

	if strings.Count(out, "define wan_interfaces = ") != 1 {
		t.Fatalf("expected a single wan_interfaces define, got:\n%s", out)
	}

	if !strings.Contains(out, `define wan_interfaces = "eth0"`) {
		t.Fatalf("expected the interface define, got:\n%s", out)
	}

	if !strings.Contains(out, `define vrf_wan_interfaces = "wan"`) {
		t.Fatalf("expected the vrf define prefixed, got:\n%s", out)
	}
}

func TestNftWireguardAllowOptIn(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "rtr",
		Interfaces: map[string]*config.Interface{"wan": {Device: "eth0", Addresses: []string{"203.0.113.2/24"}}},
		Wireguard: map[string]*config.Wireguard{
			"wg0": {ListenPort: 51900, AllowInbound: true},
			"wg1": {ListenPort: 51820, AllowInbound: true},
			"wg2": {ListenPort: 51000},
		},
	}
	out := renderNft(t, cfg)
	if !strings.Contains(out, `udp dport { 51820, 51900 } accept comment "routier: wireguard"`) {
		t.Fatalf("expected opted-in wireguard ports allowed, got:\n%s", out)
	}

	if strings.Contains(out, "51000") {
		t.Fatalf("non-opted-in wireguard port should not be allowed, got:\n%s", out)
	}
}

func TestNftContextAllows(t *testing.T) {
	cfg := &config.Config{
		Hostname: "rtr",
		Interfaces: map[string]*config.Interface{
			"wan": {Device: "eth0", Addresses: []string{"203.0.113.2/24"}},
			"lan": {Device: "eth1", Addresses: []string{"10.0.0.1/24", "2a0c:1::1/64"}, VRRP: []*config.VRRPInstance{{ID: 10, VIPs: []string{"10.0.0.1/24"}, AllowInbound: true}}},
			"dmz": {Device: "eth3", VRRP: []*config.VRRPInstance{{ID: 20, VIPs: []string{"10.0.1.1/24"}}}},
		},
		Conntrackd: &config.Conntrackd{Interface: "eth2", Port: 3780, AllowInbound: true},
		Routing: &config.Routing{
			BGP:  &config.BGP{ASN: 65000, AllowInbound: []string{"lan"}},
			OSPF: &config.OSPF{AllowInbound: []string{"lan"}},
		},
	}
	out := renderNft(t, cfg)
	if !strings.Contains(out, `iifname "eth2" udp dport 3780 accept comment "routier: conntrackd"`) {
		t.Fatalf("expected conntrackd allow, got:\n%s", out)
	}

	if !strings.Contains(out, `iifname "eth1" meta l4proto 112 accept comment "routier: vrrp"`) {
		t.Fatalf("expected vrrp allow on eth1, got:\n%s", out)
	}

	if strings.Contains(out, `iifname "eth3" meta l4proto 112`) {
		t.Fatalf("non-opted-in vrrp instance should not be allowed, got:\n%s", out)
	}

	if !strings.Contains(out, `iifname $lan_interfaces ip saddr $lan_network tcp dport 179 accept comment "routier: bgp"`) {
		t.Fatalf("expected scoped bgp v4 allow, got:\n%s", out)
	}

	if !strings.Contains(out, `iifname $lan_interfaces ip6 saddr $lan_network6 tcp dport 179 accept comment "routier: bgp"`) {
		t.Fatalf("expected scoped bgp v6 allow, got:\n%s", out)
	}

	if !strings.Contains(out, `iifname $lan_interfaces ip saddr $lan_network meta l4proto 89 accept comment "routier: ospf"`) {
		t.Fatalf("expected scoped ospf v4 allow, got:\n%s", out)
	}
}
