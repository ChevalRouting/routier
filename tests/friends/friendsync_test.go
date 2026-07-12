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
			"lan": {
				Select: "name=lan0",
				VRRP:   []*config.VRRPInstance{{ID: 10, VIPs: []string{"10.0.0.1/24"}, Priority: 100}},
			},
			"plain": {Select: "name=lan1"},
		},
		Conntrackd: &config.Conntrackd{Interface: "ha0", Address: "172.16.0.1", Port: 3780},
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

	if _, ok := out.Interfaces["lan"]; !ok {
		t.Fatal("expected vrrp interface lan replicated")
	}

	if _, ok := out.Interfaces["plain"]; ok {
		t.Fatal("non-vrrp interface should not be replicated")
	}

	if out.Conntrackd == nil {
		t.Fatal("expected conntrackd in payload")
	}

	if out.Conntrackd.Address != "172.16.0.2" {
		t.Fatalf("target address = %q", out.Conntrackd.Address)
	}

	want := map[string]bool{"172.16.0.1": true, "172.16.0.3": true}
	if len(out.Conntrackd.PeerIPs) != len(want) {
		t.Fatalf("peers = %v", out.Conntrackd.PeerIPs)
	}

	for _, p := range out.Conntrackd.PeerIPs {
		if !want[p] {
			t.Fatalf("unexpected peer %q in %v", p, out.Conntrackd.PeerIPs)
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

	if cfg.Interfaces["lan"].VRRP[0].Priority != 100 {
		t.Fatalf("source priority mutated to %d", cfg.Interfaces["lan"].VRRP[0].Priority)
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
