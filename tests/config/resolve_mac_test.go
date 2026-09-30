package configtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestResolveMACBondIdentity(t *testing.T) {
	root := t.TempDir()

	first, second := "a0:36:9f:52:da:f4", "a0:36:9f:52:da:f6"
	for _, name := range []string{"core", "isp", "eth1", "eth2"} {
		writeMACFixture(t, root, name, "address", first)
	}

	for _, name := range []string{"eth1", "eth2"} {
		writeMACFixture(t, root, name, "device/present", "")
	}

	writeMACFixture(t, root, "eth1", "bonding_slave/perm_hwaddr", first)
	writeMACFixture(t, root, "eth2", "bonding_slave/perm_hwaddr", second)
	for mac, want := range map[string]string{first: "eth1", second: "eth2"} {
		got, err := config.ResolveMACAt(root, mac)
		if err != nil || got != want {
			t.Fatalf("%s: got %q, %v; want %s", mac, got, err, want)
		}
	}

	writeMACFixture(t, root, "eth3", "device/present", "")
	writeMACFixture(t, root, "eth3", "address", first)
	if _, err := config.ResolveMACAt(root, first); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("want ambiguity, got %v", err)
	}
}

func TestResolveMembersDoesNotAccumulate(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{
		"port1": {Type: "dummy"},
		"port2": {Type: "dummy"},
		"core":  {Type: "bond", Bond: &config.Bond{Members: []string{"port1", "port2"}, Primary: "port1"}},
		"br":    {Type: "bridge", Bridge: &config.Bridge{Members: []string{"core"}}},
	}}
	for i := 0; i < 3; i++ {
		config.ResolveInterfaces(cfg)
	}

	if got := cfg.Interfaces["core"].Bond.MemberDevices; len(got) != 2 {
		t.Fatalf("accumulated bond members: %v", got)
	}

	if got := cfg.Interfaces["br"].Bridge.MemberDevices; len(got) != 1 {
		t.Fatalf("accumulated bridge members: %v", got)
	}

	cfg.Interfaces["core"].Bond.Members = []string{"port2"}
	cfg.Interfaces["core"].Bond.Primary = ""
	config.ResolveInterfaces(cfg)
	b := cfg.Interfaces["core"].Bond
	if len(b.MemberDevices) != 1 || b.MemberDevices[0] != "port2" || b.PrimaryDevice != "" {
		t.Fatalf("stale resolution: %+v", b)
	}
}

func writeMACFixture(t *testing.T, root, name, file, value string) {
	t.Helper()
	path := filepath.Join(root, name, file)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}
