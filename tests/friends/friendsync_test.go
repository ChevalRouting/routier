package friendstest

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
)

func haEnabled() *bool { b := true; return &b }

func clusterConfig() *config.Config {
	return &config.Config{
		Version: "v3.0.0",
		Interfaces: map[string]*config.Interface{
			"lan":   {Select: "name=lan0"},
			"plain": {Select: "name=lan1"},
		},
		HA: &config.HA{
			VRRP:       []*config.VRRPInstance{{ID: 10, Interface: "lan", VIPs: []string{"10.0.0.1/24"}, Priority: 100}},
			Conntrackd: &config.Conntrackd{Interface: "ha0", Address: "172.16.0.1", Port: 3780},
		},
		Friends: []*config.Friend{
			{
				Name: "peer-b", URL: "https://198.51.100.2:8080", Token: "t", Enabled: haEnabled(),
				HA: &config.FriendHA{Enabled: true, Link: &config.FriendHALink{Interface: "ha0", Address: "172.16.0.2"}},
			},
			{
				Name: "peer-c", URL: "https://198.51.100.3:8080", Token: "t", Enabled: haEnabled(),
				HA: &config.FriendHA{Enabled: true, Link: &config.FriendHALink{Interface: "ha0", Address: "172.16.0.3"}},
			},
			{Name: "peer-d", URL: "https://198.51.100.4:8080", Token: "t", Enabled: haEnabled()},
		},
	}
}

func TestBuildFriendSyncPayload(t *testing.T) {
	cfg := clusterConfig()
	target := cfg.Friends[0]

	payload := managers.BuildFriendSyncPayload(cfg, target)
	out := payload.Config

	if out.HA == nil || len(out.HA.VRRP) != 1 {
		t.Fatalf("expected one vrrp instance replicated, got %+v", out.HA)
	}

	if out.HA.VRRP[0].Interface != "lan" {
		t.Fatalf("expected vrrp on lan, got %q", out.HA.VRRP[0].Interface)
	}

	if out.Interfaces != nil {
		t.Fatal("interfaces should not be replicated for a vrrp sync")
	}

	if out.HA.Conntrackd == nil {
		t.Fatal("expected conntrackd in payload")
	}

	if out.HA.Conntrackd.Address != "172.16.0.2" {
		t.Fatalf("target address = %q", out.HA.Conntrackd.Address)
	}

	want := map[string]bool{"172.16.0.1": true, "172.16.0.3": true}
	if len(out.HA.Conntrackd.PeerIPs) != len(want) {
		t.Fatalf("peers = %v", out.HA.Conntrackd.PeerIPs)
	}

	for _, p := range out.HA.Conntrackd.PeerIPs {
		if !want[p] {
			t.Fatalf("unexpected peer %q in %v", p, out.HA.Conntrackd.PeerIPs)
		}
	}
}

func TestBuildFriendSyncPayloadDoesNotMutateSource(t *testing.T) {
	cfg := clusterConfig()
	target := cfg.Friends[0]
	target.Sync = &config.FriendSync{
		Sections:  []string{"vrrp", "conntrackd"},
		Overrides: map[string]string{"vrrp.priority": "250"},
	}

	managers.BuildFriendSyncPayload(cfg, target)

	if cfg.HA.VRRP[0].Priority != 100 {
		t.Fatalf("source priority mutated to %d", cfg.HA.VRRP[0].Priority)
	}
}

func TestSyncSectionsDefault(t *testing.T) {
	f := &config.Friend{Name: "peer-b"}
	got := managers.SyncSections(f)
	if len(got) != 2 || got[0] != "vrrp" || got[1] != "conntrackd" {
		t.Fatalf("default sections = %v", got)
	}

	f.Sync = &config.FriendSync{Sections: []string{"vrrp"}}
	if got := managers.SyncSections(f); len(got) != 1 || got[0] != "vrrp" {
		t.Fatalf("explicit sections = %v", got)
	}
}
