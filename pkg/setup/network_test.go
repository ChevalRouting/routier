package setup

import (
	"slices"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/nftables/nat"
)

func TestApplyStaticUplinkAndLAN(t *testing.T) {
	cfg := &config.Config{Hostname: "routier", Routing: &config.Routing{Static: []config.StaticRoute{{Destination: "0.0.0.0/0", Via: "198.51.100.1"}, {Destination: "10.9.0.0/16", Via: "10.0.0.2"}}}}
	err := Apply(cfg, Network{
		Hostname: "edge-1",
		Uplink:   &Uplink{Select: "eth0", Mode: "static", Address: "192.0.2.2/24", Gateway: "192.0.2.1", DNS: []string{"9.9.9.9"}},
		LANs:     []LAN{{Name: "lan", Select: "eth1", Address: "192.168.1.1/24"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Hostname != "edge-1" || !slices.Equal(cfg.Interfaces[WANInterface].Addresses, []string{"192.0.2.2/24", "slaac"}) || cfg.Interfaces["lan"].Select != "eth1" {
		t.Fatalf("interfaces %+v", cfg.Interfaces)
	}

	want := []config.StaticRoute{{Destination: "10.9.0.0/16", Via: "10.0.0.2"}, {Destination: "0.0.0.0/0", Via: "192.0.2.1", Dev: WANInterface}}
	if !slices.Equal(cfg.Routing.Static, want) {
		t.Fatalf("routes %+v", cfg.Routing.Static)
	}

	if !slices.Equal(cfg.DNS.Nameservers, []string{"9.9.9.9"}) {
		t.Fatalf("dns %+v", cfg.DNS)
	}

	specs := nat.Parse(cfg.Nftables)
	if len(specs) != 1 || specs[0].Source != "$lan_network" || specs[0].Out != "$wan_interfaces" {
		t.Fatalf("nat %+v", specs)
	}

	if errs := config.Validate(cfg, false); len(errs) != 0 {
		t.Fatal(errs)
	}
}

func TestApplyTwiceKeepsOneSetOfRules(t *testing.T) {
	cfg := &config.Config{}
	network := Network{Hostname: "edge-1", Uplink: &Uplink{Select: "eth0", Mode: "dhcp"}, LANs: []LAN{{Name: "lan", Select: "eth1", Address: "192.168.1.1/24"}}}
	for i := 0; i < 2; i++ {
		if err := Apply(cfg, network); err != nil {
			t.Fatal(err)
		}
	}

	if len(cfg.Nftables.Chains["forward"].Managed) != 3 || len(nat.Parse(cfg.Nftables)) != 1 {
		t.Fatalf("rules %+v", cfg.Nftables.Chains["forward"].Managed)
	}

	if !slices.Equal(cfg.Interfaces[WANInterface].Addresses, []string{"dhcp", "slaac"}) {
		t.Fatalf("wan %+v", cfg.Interfaces[WANInterface])
	}
}

func TestApplyRefusesBadInput(t *testing.T) {
	for _, network := range []Network{
		{},
		{Hostname: "a", Uplink: &Uplink{Mode: "dhcp"}},
		{Hostname: "a", Uplink: &Uplink{Select: "eth0", Mode: "static", Address: "192.0.2.2", Gateway: "192.0.2.1"}},
		{Hostname: "a", LANs: []LAN{{Name: "wan", Select: "eth1", Address: "192.168.1.1/24"}}},
		{Hostname: "a", LANs: []LAN{{Name: "lan", Select: "eth1", Address: "bogus"}}},
	} {
		if err := Apply(&config.Config{}, network); err == nil {
			t.Errorf("network %+v accepted", network)
		}
	}
}
