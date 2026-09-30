package simple

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestReplaceNetworksBuildsDualStackDHCPAndRADVD(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{}}
	values := []Network{{
		Name: "lan", Select: "name=eth1", Addresses: []string{"192.168.1.1/24", "fd00:1::1/64"},
		ManageDHCP: true, Pool: "192.168.1.100-192.168.1.200",
		ManageDHCP6: true, PoolV6: "fd00:1::100-fd00:1::ffff", DNS: []string{"192.168.1.1", "fd00:1::1"},
	}}

	got, err := replaceNetworks(cfg, values)
	if err != nil {
		t.Fatal(err)
	}
	if got.DHCP == nil || len(got.DHCP.Subnets4) != 1 || len(got.DHCP.Subnets6) != 1 {
		t.Fatalf("unexpected DHCP config: %#v", got.DHCP)
	}
	if got.DHCP.Subnets6[0].Subnet != "fd00:1::/64" {
		t.Fatalf("unexpected DHCPv6 subnet: %q", got.DHCP.Subnets6[0].Subnet)
	}
	ra := got.Routing.RADVD.Interfaces["lan"]
	if ra == nil || !ra.AdvManagedFlag || len(ra.Prefixes) != 1 || ra.Prefixes[0].Prefix != "fd00:1::/64" {
		t.Fatalf("unexpected RADVD config: %#v", ra)
	}
	if projected := inspect(got).Networks[0]; !projected.ManageDHCP || !projected.ManageDHCP6 {
		t.Fatalf("dual-stack DHCP did not round trip: %#v", projected)
	}
}

func TestSimpleDNSPreservesHostedZones(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{"lan": {Addresses: []string{"192.168.1.1/24"}}}}
	zone := config.DNSZone{Name: "home.arpa", Records: []config.DNSRecord{{Name: "router", Type: "A", Value: "192.168.1.1"}}}
	got, err := replaceDNS(cfg, DNS{Enabled: true, Upstreams: []string{"1.1.1.1"}, Networks: []string{"lan"}, Zones: []config.DNSZone{zone}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.DNS.Server.Zones) != 1 || got.DNS.Server.Zones[0].Name != zone.Name {
		t.Fatalf("zone was not written: %#v", got.DNS.Server.Zones)
	}
	projected := inspectDNS(got, []Network{{Name: "lan", Addresses: []string{"192.168.1.1/24"}}})
	if len(projected.Zones) != 1 || projected.Zones[0].Records[0].Value != "192.168.1.1" {
		t.Fatalf("zone did not round trip: %#v", projected.Zones)
	}
}

func TestSimpleConfigPreservesMonitoringProbes(t *testing.T) {
	value := Config{System: System{Hostname: "router"}, Monitoring: &config.MonitoringConfig{Probes: []config.PingProbe{{Name: "internet", Target: "1.1.1.1", Interval: 300, Timeout: 2500}}}}
	got, err := build(value)
	if err != nil {
		t.Fatal(err)
	}
	if got.Monitoring == nil || len(got.Monitoring.Probes) != 1 || got.Monitoring.Probes[0].Target != "1.1.1.1" {
		t.Fatalf("probe was not preserved: %#v", got.Monitoring)
	}
}

func TestSimpleReplaceSectionPreservesAdvancedConfig(t *testing.T) {
	cfg := &config.Config{
		Hostname: "old",
		Interfaces: map[string]*config.Interface{
			"core":  {Type: "bond", Bond: &config.Bond{Members: []string{"core1", "core2"}, Mode: "802.3ad"}},
			"core1": {Select: "mac(a0:36:9f:52:da:f4)"},
			"core2": {Select: "mac(a0:36:9f:52:da:f6)"},
		},
		Routing: &config.Routing{BGP: &config.BGP{ASN: 65000}},
		VRFs:    map[string]*config.VRFConfig{"blue": {Table: 100}},
	}

	got, err := Layer{}.ReplaceSection(cfg, "hostname", []byte(`"new"`))
	if err != nil {
		t.Fatal(err)
	}

	if got.Hostname != "new" {
		t.Fatalf("hostname was not updated: %q", got.Hostname)
	}

	core := got.Interfaces["core"]
	if core == nil || core.Bond == nil {
		t.Fatalf("advanced bond/vlan interface was dropped: %#v", got.Interfaces)
	}

	if got.Routing == nil || got.Routing.BGP == nil {
		t.Fatalf("advanced routing was dropped: %#v", got.Routing)
	}

	if got.VRFs["blue"] == nil {
		t.Fatalf("advanced vrfs were dropped: %#v", got.VRFs)
	}
}
