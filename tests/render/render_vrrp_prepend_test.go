package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func vrrpPrependConfig(neighbors []string) *config.Config {
	return &config.Config{
		Hostname:   "rt1",
		Interfaces: map[string]*config.Interface{"lan": {Device: "eth0"}},
		Routing: &config.Routing{BGP: &config.BGP{
			ASN:      65010,
			RouterID: "10.0.0.1",
			Neighbors: []config.BGPNeighbor{
				{Address: "192.0.2.1", RemoteASN: 64500, AddressFamilies: map[string]*config.BGPNeighborAF{"ipv4-unicast": {RouteMapOut: "TRANSIT-OUT"}}},
				{Address: "2001:db8::1", RemoteASN: 64501},
				{Address: "10.0.0.2", RemoteASN: 65010},
			},
			RouteMaps: map[string][]config.RouteMapEntry{"TRANSIT-OUT": {{Seq: 10, Action: "permit"}}},
		}},
		HA: &config.HA{VRRP: []*config.VRRPInstance{{
			ID:         51,
			Interface:  "lan",
			VIPs:       []string{"10.0.0.254/24"},
			BGPPrepend: &config.VRRPBGPPrepend{Neighbors: neighbors},
		}}},
	}
}

func TestRenderVRRPPrependWrapsEBGPNeighbors(t *testing.T) {
	m := keaOutputs(t, vrrpPrependConfig(nil))
	frr := m["/etc/frr/frr.conf"]

	wantLines(t, "frr.conf", frr,
		"route-map ROUTIER-VRRP-51 permit 10\n set as-path prepend 65010 65010 65010\nexit",
		"neighbor 192.0.2.1 route-map ROUTIER-OUT-192_0_2_1-ipv4-unicast out",
		"route-map ROUTIER-OUT-192_0_2_1-ipv4-unicast permit 100\n call ROUTIER-VRRP-51\n on-match next\nexit",
		"route-map ROUTIER-OUT-192_0_2_1-ipv4-unicast permit 1000\n call TRANSIT-OUT\n on-match next\nexit",
		"route-map ROUTIER-OUT-192_0_2_1-ipv4-unicast permit 9999\nexit",
		"neighbor 2001:db8::1 route-map ROUTIER-OUT-2001_db8__1-ipv6-unicast out",
		"route-map ROUTIER-OUT-2001_db8__1-ipv6-unicast deny 9999\nexit",
	)

	if strings.Contains(frr, "neighbor 192.0.2.1 route-map TRANSIT-OUT out") {
		t.Errorf("covered neighbor kept its own out map instead of the wrapper:\n%s", frr)
	}

	if strings.Contains(frr, "ROUTIER-OUT-10_0_0_2-ipv4-unicast permit 100\n call ROUTIER-VRRP-51") {
		t.Errorf("ibgp neighbor must not be prepended by default:\n%s", frr)
	}

	wantLines(t, "keepalived.conf", m["/etc/keepalived/keepalived.conf"], "/routier vrrp-notify\"")
}

func TestRenderVRRPPrependListedNeighbors(t *testing.T) {
	frr := keaOutputs(t, vrrpPrependConfig([]string{"10.0.0.2"}))["/etc/frr/frr.conf"]

	if !strings.Contains(frr, "route-map ROUTIER-VRRP-51 permit 10") {
		t.Fatalf("prepend map missing:\n%s", frr)
	}

	if strings.Contains(frr, "ROUTIER-OUT-192_0_2_1-ipv4-unicast permit 100\n call ROUTIER-VRRP-51") {
		t.Errorf("only listed neighbors must be prepended:\n%s", frr)
	}
}

func TestRenderBGPPolicyDeniesWithoutUserRules(t *testing.T) {
	frr := keaOutputs(t, vrrpPrependConfig(nil))["/etc/frr/frr.conf"]

	wantLines(t, "frr.conf", frr,
		"neighbor 10.0.0.2 route-map ROUTIER-IN-10_0_0_2-ipv4-unicast in",
		"neighbor 10.0.0.2 route-map ROUTIER-OUT-10_0_0_2-ipv4-unicast out",
		"route-map ROUTIER-IN-10_0_0_2-ipv4-unicast deny 9999\nexit",
		"route-map ROUTIER-OUT-10_0_0_2-ipv4-unicast deny 9999\nexit",
	)
}

func TestRenderVRRPPrependScopedToPrefixes(t *testing.T) {
	cfg := vrrpPrependConfig(nil)
	cfg.HA.VRRP[0].BGPPrepend.Prefixes = []string{"203.0.113.0/24", "198.51.100.7/32", "2001:db8:100::/48"}
	frr := keaOutputs(t, cfg)["/etc/frr/frr.conf"]

	wantLines(t, "frr.conf", frr,
		"ip prefix-list ROUTIER-VRRP-51-V4 seq 5 permit 203.0.113.0/24 le 32",
		"ip prefix-list ROUTIER-VRRP-51-V4 seq 10 permit 198.51.100.7/32\n",
		"ipv6 prefix-list ROUTIER-VRRP-51-V6 seq 5 permit 2001:db8:100::/48 le 128",
		"route-map ROUTIER-VRRP-51 permit 10\n match ip address prefix-list ROUTIER-VRRP-51-V4\n set as-path prepend 65010 65010 65010\nexit",
		"route-map ROUTIER-VRRP-51 permit 20\n match ipv6 address prefix-list ROUTIER-VRRP-51-V6\n set as-path prepend 65010 65010 65010\nexit",
		"route-map ROUTIER-VRRP-51 permit 100\nexit",
	)
}

func TestRenderVRRPTracksDevices(t *testing.T) {
	cfg := vrrpPrependConfig(nil)
	cfg.HA.VRRP[0].BGPPrepend = nil
	cfg.HA.VRRP[0].TrackInterfaces = []string{"wan"}
	cfg.HA.VRRP[0].TrackDevices = []string{"eth9"}
	keepalived := keaOutputs(t, cfg)["/etc/keepalived/keepalived.conf"]
	if !strings.Contains(keepalived, "track_interface {\n        eth9") && !strings.Contains(keepalived, "eth9") {
		t.Fatalf("keepalived.conf does not track eth9:\n%s", keepalived)
	}

	if strings.Contains(keepalived, " wan\n") {
		t.Fatalf("keepalived.conf tracks the logical name:\n%s", keepalived)
	}
}
