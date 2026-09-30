package managers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRollbackConfigResolvesDevices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "last-applied.yml")
	data := `version: v3.0.0
interfaces:
  uplink:
    select: eth0
  isp:
    type: vlan
    select: uplink
    vlan:
      id: 100
  lan:
    type: bridge
    bridge:
      members: [eth1, eth2]
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadRollbackConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"uplink": "eth0", "isp": "isp", "lan": "lan"} {
		if got := cfg.Interfaces[name].Device; got != want {
			t.Fatalf("%s device=%q, want %q", name, got, want)
		}
	}
	members := cfg.Interfaces["lan"].Bridge.MemberDevices
	if len(members) != 2 || members[0] != "eth1" || members[1] != "eth2" {
		t.Fatalf("unresolved bridge members: %v", members)
	}
}
