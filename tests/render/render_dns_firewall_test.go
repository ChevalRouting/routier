package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestDNSDisableAutoRules(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "rtr",
		Interfaces: map[string]*config.Interface{"lan": {Device: "eth1", Addresses: []string{"10.0.0.1/24", "2001:db8::1/64"}}},
		VPN:        config.VPNConfig{Wireguard: map[string]*config.Wireguard{"wg0": {ListenPort: 51820, AllowInbound: true}}},
		Routing: &config.Routing{
			BGP:   &config.BGP{ASN: 65000, AllowInbound: []string{"lan"}},
			OSPF:  &config.OSPF{AllowInbound: []string{"lan"}},
			OSPF6: &config.OSPF6{AllowInbound: []string{"lan"}},
		},
		HA: &config.HA{
			Conntrackd: &config.Conntrackd{Interface: "lan", AllowInbound: true},
			VRRP:       []*config.VRRPInstance{{ID: 1, Interface: "lan", VIPs: []string{"10.0.0.2/24"}, AllowInbound: true}},
		},
		DNS:      &config.DNS{Server: &config.DNSServer{Enabled: true, Listen: []string{"iface(lan)"}, AllowFrom: []string{"10.0.0.0/24", "2001:db8::/64"}, AllowInbound: []string{"lan"}}},
		Nftables: &config.NftablesConfig{Chains: map[string]*config.NftChain{"input": {Policy: "drop", Rules: "tcp dport 443 accept"}}},
	}
	for _, disabled := range []bool{false, true} {
		cfg.DNS.Server.DisableAutoRules = disabled
		output := renderNft(t, cfg)
		if present := strings.Contains(output, `comment "routier: dns"`); present == disabled {
			t.Errorf("DNS disable_auto_rules=%v: unexpected DNS allow rules", disabled)
		}

		for _, want := range []string{`comment "routier: wireguard"`, `comment "routier: bgp"`, `comment "routier: ospf"`, `comment "routier: ospf6"`, `comment "routier: vrrp"`, `comment "routier: conntrackd"`, "tcp dport 443 accept", "policy drop;"} {
			if !strings.Contains(output, want) {
				t.Errorf("DNS disable_auto_rules=%v: missing %q", disabled, want)
			}
		}

		if len(cfg.DNS.Server.AllowInbound) != 1 {
			t.Fatal("DNS interface selection was discarded")
		}
	}
}
