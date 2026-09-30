package rendertest

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/tests/testkit"
)

func queryAddrConfig(t *testing.T, listen string) *config.Config {
	t.Helper()

	yaml := `version: v3.0.0
hostname: gw
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.1/24"]
dns:
  server:
    enabled: true
    listen: ` + listen + `
    allow_from: [10.0.0.0/24]
    allow_inbound: [lan]
`

	cfg, err := config.Load(testkit.WriteConfig(t, t.TempDir(), yaml))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	return cfg
}

func TestDNSQueryAddressSpecificListen(t *testing.T) {
	got := render.DNSQueryAddress(queryAddrConfig(t, "[iface(lan)]"))
	if got != "10.0.0.1" {
		t.Fatalf("query address = %q, want the listen address 10.0.0.1", got)
	}
}

func TestDNSQueryAddressUnspecifiedV4(t *testing.T) {
	got := render.DNSQueryAddress(queryAddrConfig(t, "[0.0.0.0]"))
	if got != "127.0.0.1" {
		t.Fatalf("query address = %q, want 127.0.0.1", got)
	}
}

func TestDNSQueryAddressUnspecifiedV6(t *testing.T) {
	got := render.DNSQueryAddress(queryAddrConfig(t, `["::"]`))
	if got != "::1" {
		t.Fatalf("query address = %q, want ::1", got)
	}
}

func TestDNSQueryAddressPrefersLoopback(t *testing.T) {
	got := render.DNSQueryAddress(queryAddrConfig(t, "[iface(lan), 127.0.0.1]"))
	if got != "127.0.0.1" {
		t.Fatalf("query address = %q, want the loopback listen address", got)
	}
}
