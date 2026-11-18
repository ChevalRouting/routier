package config

import (
	"errors"
	"testing"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
)

func TestScopeFriendPayloadRejectsUnmanagedSections(t *testing.T) {
	sender := &cfgpkg.Friend{Name: "edge-b", Hostname: "edge-b"}
	local := &cfgpkg.Config{Interfaces: map[string]*cfgpkg.Interface{"wan": {Type: "ethernet"}}}

	cases := map[string]*managers.PushPayload{
		"files":           {Config: &cfgpkg.Config{}, Files: map[string]string{"/etc/routier/templates/x": "y"}},
		"untagged iface":  {Config: &cfgpkg.Config{Interfaces: map[string]*cfgpkg.Interface{"gre0": {Type: "gre"}}}},
		"overwrite iface": {Config: &cfgpkg.Config{Interfaces: map[string]*cfgpkg.Interface{"wan": {Friend: "edge-b"}}}},
		"wg hook": {Config: &cfgpkg.Config{VPN: cfgpkg.VPNConfig{Wireguard: map[string]*cfgpkg.Wireguard{
			"wg-b": {Friend: "edge-b", PostUp: []string{"id"}},
		}}}},
		"conntrackd": {Config: &cfgpkg.Config{HA: &cfgpkg.HA{Conntrackd: &cfgpkg.Conntrackd{}}}},
	}

	for name, payload := range cases {
		if err := scopeFriendPayload(local, sender, payload); !errors.Is(err, errFriendScope) {
			t.Errorf("%s: got %v, want errFriendScope", name, err)
		}
	}
}

func TestScopeFriendPayloadKeepsTaggedSections(t *testing.T) {
	sender := &cfgpkg.Friend{Name: "edge-b", Hostname: "edge-b"}
	local := &cfgpkg.Config{}
	payload := &managers.PushPayload{Config: &cfgpkg.Config{
		Users:      map[string]*cfgpkg.User{"evil": {}},
		Interfaces: map[string]*cfgpkg.Interface{"gre-b": {Type: "gre", Friend: "edge-b"}},
	}}

	if err := scopeFriendPayload(local, sender, payload); err != nil {
		t.Fatal(err)
	}

	if payload.Config.Users != nil || payload.Config.Interfaces["gre-b"] == nil {
		t.Fatalf("unexpected scoped payload: %+v", payload.Config)
	}
}

func TestScopeFriendPayloadTrustsManageFriends(t *testing.T) {
	sender := &cfgpkg.Friend{Name: "edge-b", Manage: true}
	payload := &managers.PushPayload{Config: &cfgpkg.Config{Users: map[string]*cfgpkg.User{"admin": {}}}, Files: map[string]string{"a": "b"}}

	if err := scopeFriendPayload(&cfgpkg.Config{}, sender, payload); err != nil || payload.Config.Users == nil {
		t.Fatalf("manage friend payload altered: %v", err)
	}
}
