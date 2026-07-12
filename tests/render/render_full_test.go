package rendertest

import (
	"github.com/ChevalRouting/routier/tests/harness"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func renderByDest(t *testing.T, yaml string) map[string]string {
	t.Helper()

	cfg, err := config.Load(harness.WriteConfig(t, t.TempDir(), yaml))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	outputs, err := render.All(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	byDest := make(map[string]string, len(outputs))
	for _, o := range outputs {
		byDest[o.Dest] = o.Content
	}

	return byDest
}

func TestRenderFullConfig(t *testing.T) {
	out := renderByDest(t, harness.FullConfig)

	checks := map[string][]string{
		"/etc/nftables.d/routier.nft":         {"table inet routier"},
		"/etc/frr/frr.conf":                   {"router bgp 65010", "neighbor 198.51.100.1"},
		"/etc/frr/daemons":                    {"bgpd"},
		"/etc/sysctl.d/99-routier.conf":       {"net.ipv4.ip_forward"},
		"/etc/keepalived/keepalived.conf":     {"vrrp_instance", "10.0.0.254"},
		"/etc/conntrackd/conntrackd.conf":     {"10.0.0.3"},
		"/etc/ssh/sshd_config.d/routier.conf": {"2222"},
		"/etc/wireguard/wg0.conf":             {"ListenPort = 51820", "10.10.0.2/32"},
	}

	for dest, wants := range checks {
		content, ok := out[dest]
		if !ok {
			t.Errorf("missing rendered output for %s", dest)
			continue
		}

		for _, w := range wants {
			if !strings.Contains(content, w) {
				t.Errorf("%s missing %q in:\n%s", dest, w, content)
			}
		}
	}
}
