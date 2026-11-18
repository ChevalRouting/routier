package managers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRollbackConfigResolvesDevices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "last-applied.yml")
	data := `version: v5.0.0
interfaces:
  uplink:
    select: lo
  isp:
    type: vlan
    select: uplink
    vlan:
      id: 100
  spare:
    type: dummy
  lan:
    type: bridge
    bridge:
      members: [uplink, spare]
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadRollbackConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]string{"uplink": "lo", "isp": "isp", "spare": "spare", "lan": "lan"} {
		if got := cfg.Interfaces[name].Device; got != want {
			t.Fatalf("%s device=%q, want %q", name, got, want)
		}
	}

	members := cfg.Interfaces["lan"].Bridge.MemberDevices
	if len(members) != 2 || members[0] != "lo" || members[1] != "spare" {
		t.Fatalf("unresolved bridge members: %v", members)
	}
}
