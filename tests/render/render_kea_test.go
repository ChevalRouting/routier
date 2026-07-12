package rendertest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func keaOutputs(t *testing.T, cfg *config.Config) map[string]string {
	t.Helper()
	outs, err := render.All(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	m := map[string]string{}
	for _, o := range outs {
		m[o.Dest] = o.Content
	}

	return m
}

func TestRenderKeaLocal(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "gw",
		Interfaces: map[string]*config.Interface{"lan": {Device: "eth1"}},
		DHCP: &config.DHCP{
			Enabled: true,
			Subnets4: []config.KeaSubnet{{
				Subnet:        "10.0.0.0/24",
				Interface:     "lan",
				Pools:         []string{"10.0.0.100-10.0.0.200"},
				Gateway:       "10.0.0.1",
				DNS:           []string{"1.1.1.1", "9.9.9.9"},
				ValidLifetime: 3600,
				Reservations: []config.KeaReservation{
					{Hostname: "printer", HWAddress: "aa:bb:cc:dd:ee:ff", IPAddress: "10.0.0.5"},
				},
			}},
			Subnets6: []config.KeaSubnet{{
				Subnet: "2001:db8::/64",
				Pools:  []string{"2001:db8::1000-2001:db8::2000"},
				DNS:    []string{"2001:db8::53"},
			}},
		},
	}

	m := keaOutputs(t, cfg)

	d4, ok := m["/etc/kea/kea-dhcp4.conf"]
	if !ok {
		t.Fatal("kea-dhcp4.conf not rendered")
	}

	var wrap4 struct {
		Dhcp4 map[string]json.RawMessage `json:"Dhcp4"`
	}
	if err := json.Unmarshal([]byte(d4), &wrap4); err != nil {
		t.Fatalf("dhcp4 is not valid json: %v\n%s", err, d4)
	}

	if !strings.Contains(d4, `"eth1"`) {
		t.Errorf("dhcp4 did not resolve per-subnet interface 'lan' to device 'eth1':\n%s", d4)
	}

	for _, want := range []string{"10.0.0.0/24", "10.0.0.100-10.0.0.200", "routers", "10.0.0.1", "domain-name-servers", "aa:bb:cc:dd:ee:ff", "lease_cmds", "control-socket", `"unix"`, "kea-dhcp4-ctrl.sock", `"interface": "eth1"`, `"valid-lifetime": 3600`, "output-options"} {
		if !strings.Contains(d4, want) {
			t.Errorf("dhcp4 missing %q", want)
		}
	}

	d6, ok := m["/etc/kea/kea-dhcp6.conf"]
	if !ok {
		t.Fatal("kea-dhcp6.conf not rendered")
	}

	for _, want := range []string{"2001:db8::/64", "dns-servers", "control-socket", "kea-dhcp6-ctrl.sock", `"*"`} {
		if !strings.Contains(d6, want) {
			t.Errorf("dhcp6 missing %q:\n%s", want, d6)
		}
	}

	if _, ok := m["/etc/kea/kea-ctrl-agent.conf"]; ok {
		t.Error("control agent should no longer be rendered (kea 3.0 uses per-server http sockets)")
	}
}

func TestRenderKeaSkippedForRemote(t *testing.T) {
	cfg := &config.Config{
		Hostname: "gw",
		DHCP: &config.DHCP{
			Enabled:      true,
			ControlAgent: &config.KeaControlAgent{URL: "http://10.9.9.9:8000/"},
			Subnets4:     []config.KeaSubnet{{Subnet: "10.0.0.0/24"}},
		},
	}

	m := keaOutputs(t, cfg)
	if _, ok := m["/etc/kea/kea-dhcp4.conf"]; ok {
		t.Error("local kea rendered for a remote control agent")
	}
}

func TestRenderKeaDisabled(t *testing.T) {
	cfg := &config.Config{Hostname: "gw", DHCP: &config.DHCP{Enabled: false, Subnets4: []config.KeaSubnet{{Subnet: "10.0.0.0/24"}}}}
	m := keaOutputs(t, cfg)
	if _, ok := m["/etc/kea/kea-ctrl-agent.conf"]; ok {
		t.Error("kea rendered while disabled")
	}
}
