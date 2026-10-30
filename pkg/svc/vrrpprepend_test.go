package svc

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

const prependRunning = `frr version 10.2
!
router bgp 65010
 neighbor 192.0.2.1 remote-as 64500
exit
!
route-map ROUTIER-VRRP-51 permit 10
 set as-path prepend 65010 65010
exit
!
route-map ROUTIER-VRRP-52 permit 10
exit
!
`

const scopedRunning = `route-map ROUTIER-VRRP-51 permit 10
 match ip address prefix-list ROUTIER-VRRP-51-V4
 set as-path prepend 65010
exit
!
route-map ROUTIER-VRRP-51 permit 20
 match ipv6 address prefix-list ROUTIER-VRRP-51-V6
exit
!
route-map ROUTIER-VRRP-51 permit 100
exit
`

func prependConfig() *config.Config {
	return &config.Config{
		Routing: &config.Routing{BGP: &config.BGP{ASN: 65010, Neighbors: []config.BGPNeighbor{{Address: "192.0.2.1", RemoteASN: 64500}}}},
		HA: &config.HA{VRRP: []*config.VRRPInstance{
			{ID: 51, Interface: "lan", VIPs: []string{"10.0.0.1/24"}, BGPPrepend: &config.VRRPBGPPrepend{Count: 2}},
			{ID: 52, Interface: "wan", VIPs: []string{"198.51.100.1"}, BGPPrepend: &config.VRRPBGPPrepend{}},
		}},
	}
}

type vtyshRecorder struct {
	running string
	calls   []string
}

func (r *vtyshRecorder) run(_ context.Context, name string, args ...string) ([]byte, error) {
	call := name + " " + strings.Join(args, " | ")
	if call == "vtysh -c | show running-config bgpd" {
		return []byte(r.running), nil
	}

	r.calls = append(r.calls, call)
	return nil, nil
}

func fakeVtysh(t *testing.T, running string) *[]string {
	t.Helper()

	r := &vtyshRecorder{running: running}
	t.Cleanup(SetCommandRunner(r.run))

	return &r.calls
}

func TestBGPPrependHook(t *testing.T) {
	cfg := prependConfig()
	calls := fakeVtysh(t, prependRunning)

	if err := RunVRRPHooks(cfg, map[string]string{"VI_lan_51": "MASTER", "VI_wan_52": "BACKUP", "VI_other_9": "MASTER"}); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"vtysh -c | configure terminal | -c | route-map ROUTIER-VRRP-51 permit 10 | -c | no set as-path prepend",
		"vtysh -c | configure terminal | -c | route-map ROUTIER-VRRP-52 permit 10 | -c | set as-path prepend 65010 65010 65010",
	}
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("vtysh calls:\n%s\nwant:\n%s", strings.Join(*calls, "\n"), strings.Join(want, "\n"))
	}

	*calls = nil
	if err := RunVRRPHooks(cfg, map[string]string{}); err != nil || len(*calls) != 0 {
		t.Fatalf("unknown states must leave the rendered prepend alone: %v %v", err, *calls)
	}
}

func TestBGPPrependHookInSync(t *testing.T) {
	calls := fakeVtysh(t, prependRunning)

	if err := RunVRRPHooks(prependConfig(), map[string]string{"VI_lan_51": "BACKUP", "VI_wan_52": "MASTER"}); err != nil {
		t.Fatal(err)
	}

	if len(*calls) != 0 {
		t.Fatalf("in-sync route-maps must not be reconfigured: %v", *calls)
	}
}

func TestBGPPrependHookRouteMapMissing(t *testing.T) {
	calls := fakeVtysh(t, "frr version 10.2\n!\n")

	err := RunVRRPHooks(prependConfig(), map[string]string{"VI_lan_51": "MASTER"})
	if err == nil || !strings.Contains(err.Error(), "not loaded in bgpd") {
		t.Fatalf("missing route-map must fail so the next sync retries: %v", err)
	}

	if len(*calls) != 0 {
		t.Fatalf("no vtysh configure expected: %v", *calls)
	}
}

func TestBGPPrependHookScopedEntries(t *testing.T) {
	cfg := &config.Config{
		Routing: &config.Routing{BGP: &config.BGP{ASN: 65010, Neighbors: []config.BGPNeighbor{{Address: "192.0.2.1", RemoteASN: 64500}}}},
		HA: &config.HA{VRRP: []*config.VRRPInstance{
			{ID: 51, Interface: "lan", VIPs: []string{"10.0.0.1"}, BGPPrepend: &config.VRRPBGPPrepend{Count: 1, Prefixes: []string{"203.0.113.0/24", "2001:db8::/48"}}},
		}},
	}

	calls := fakeVtysh(t, scopedRunning)
	if err := RunVRRPHooks(cfg, map[string]string{"VI_lan_51": "MASTER"}); err != nil {
		t.Fatal(err)
	}

	want := "vtysh -c | configure terminal | -c | route-map ROUTIER-VRRP-51 permit 10 | -c | no set as-path prepend"
	if len(*calls) != 1 || (*calls)[0] != want {
		t.Fatalf("vtysh calls: %v\nwant: %s", *calls, want)
	}

	*calls = nil
	if err := RunVRRPHooks(cfg, map[string]string{"VI_lan_51": "BACKUP"}); err != nil {
		t.Fatal(err)
	}

	want = "vtysh -c | configure terminal | -c | route-map ROUTIER-VRRP-51 permit 20 | -c | set as-path prepend 65010"
	if len(*calls) != 1 || (*calls)[0] != want {
		t.Fatalf("vtysh calls: %v\nwant: %s", *calls, want)
	}
}

func TestSyncVRRPHooksFollowsHeldVIPs(t *testing.T) {
	prevAddrs := localAddrs
	localAddrs = func() ([]net.Addr, error) {
		return []net.Addr{&net.IPNet{IP: net.ParseIP("10.0.0.1"), Mask: net.CIDRMask(24, 32)}}, nil
	}
	t.Cleanup(func() { localAddrs = prevAddrs })

	calls := fakeVtysh(t, prependRunning)
	if err := SyncVRRPHooks(prependConfig()); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"vtysh -c | configure terminal | -c | route-map ROUTIER-VRRP-51 permit 10 | -c | no set as-path prepend",
		"vtysh -c | configure terminal | -c | route-map ROUTIER-VRRP-52 permit 10 | -c | set as-path prepend 65010 65010 65010",
	}
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("vtysh calls:\n%s\nwant:\n%s", strings.Join(*calls, "\n"), strings.Join(want, "\n"))
	}
}
