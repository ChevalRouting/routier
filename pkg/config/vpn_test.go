package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const legacyVPNConfig = `version: v3.0.0
hostname: vpn-router
wireguard:
  wg0:
    private_key: preserved-key
    addresses: [10.0.0.1/24]
    peers:
      - public_key: preserved-peer
        allowed_ips: [10.0.0.2/32]
tunnels:
  gre0:
    mode: gre
    local: 192.0.2.1
    remote: 192.0.2.2
    addresses: [10.1.0.1/30]
`

func TestMigrateVPNPreservesDefinitions(t *testing.T) {
	cfg, err := LoadBytes([]byte(legacyVPNConfig))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Version != CurrentVersion || cfg.VPN.Wireguard["wg0"].PrivateKey != "preserved-key" || cfg.Interfaces["gre0"].Remote != "192.0.2.2" || cfg.Interfaces["gre0"].Type != "tunnel" {
		t.Fatalf("migration lost VPN definitions: %+v", cfg)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	vpn, _ := raw["vpn"].(map[string]any)
	if vpn["wireguard"] == nil || vpn["tunnels"] != nil || raw["wireguard"] != nil || raw["tunnels"] != nil || raw["interfaces"] == nil {
		t.Fatalf("expected canonical nested VPN schema: %s", data)
	}

	if _, changed, err := MigrateBytes(data); err != nil || changed {
		t.Fatalf("canonical VPN migration must be idempotent: changed=%v err=%v", changed, err)
	}
}

func TestMigrateVPNRejectsAmbiguousDefinitions(t *testing.T) {
	data := "version: v3.0.0\nwireguard: {}\nvpn:\n  wireguard: {}\n"
	if _, err := LoadBytes([]byte(data)); err == nil || !strings.Contains(err.Error(), "both wireguard and vpn.wireguard") {
		t.Fatalf("want conflict error, got %v", err)
	}
}

func TestLegacyVPNJSONMigration(t *testing.T) {
	var cfg Config
	data := `{"version":"v3.0.0","wireguard":{"wg0":{"private_key":"preserved"}},"tunnels":{"gre0":{"mode":"gre"}}}`
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Version != CurrentVersion || cfg.VPN.Wireguard["wg0"].PrivateKey != "preserved" || cfg.Interfaces["gre0"].Mode != "gre" {
		t.Fatalf("legacy JSON lost VPN definitions: %+v", cfg)
	}
}

func TestVPNRejectsProtocolNameCollision(t *testing.T) {
	cfg := &Config{Hostname: "router", Interfaces: map[string]*Interface{"vpn0": {Type: "tunnel", Mode: "gre", Local: "192.0.2.1"}}, VPN: VPNConfig{
		Wireguard: map[string]*Wireguard{"vpn0": {PrivateKey: "test"}},
	}}
	for _, err := range Validate(cfg, false) {
		if strings.Contains(err.Error(), "configured in both vpn.wireguard and interfaces") {
			return
		}
	}

	t.Fatal("VPN protocols must not claim the same interface name")
}

func TestMigrateNestedTunnelsToInterfaces(t *testing.T) {
	data := "version: v4.0.0\ninterfaces:\n  lan:\n    type: dummy\nvpn:\n  wireguard:\n    wg0:\n      listen_port: 51820\n  tunnels:\n    gre-office:\n      mode: gre\n      local: 192.0.2.1\n      remote: 192.0.2.2\n      ttl: 64\n      mtu: 1400\n      friend: peer\n      addresses: [10.1.0.1/30]\n"
	cfg, err := LoadBytes([]byte(data))
	if err != nil {
		t.Fatal(err)
	}

	iface := cfg.Interfaces["gre-office"]
	if cfg.Version != CurrentVersion || iface.Type != "tunnel" || iface.Mode != "gre" || iface.TTL != 64 || iface.MTU != 1400 || iface.Friend != "peer" || len(iface.Addresses) != 1 || cfg.Interfaces["lan"] == nil || cfg.VPN.Wireguard["wg0"] == nil {
		t.Fatalf("migration lost interface or VPN settings: %+v", cfg)
	}

	ResolveInterfaces(cfg)
	if iface.Device != "gre-office" {
		t.Fatalf("new tunnel must resolve before the kernel interface exists: %+v", iface)
	}

	for _, err := range Validate(cfg, false) {
		if strings.Contains(err.Error(), "interfaces.gre-office") {
			t.Fatal(err)
		}
	}
}

func TestTunnelMigrationRejectsInterfaceCollision(t *testing.T) {
	data := "version: v4.0.0\ninterfaces:\n  gre0:\n    type: dummy\nvpn:\n  tunnels:\n    gre0:\n      mode: gre\n"
	if _, err := LoadBytes([]byte(data)); err == nil || !strings.Contains(err.Error(), "both interfaces.gre0 and vpn.tunnels.gre0") {
		t.Fatalf("want interface collision error, got %v", err)
	}
}

func TestTunnelAliasPreservesInterfaceSettings(t *testing.T) {
	cfg := &Config{Interfaces: map[string]*Interface{
		"lan":  {Type: "dummy"},
		"gre0": {Type: "tunnel", Mode: "gre", VRF: "blue", Device: "gre0"},
	}}
	if err := cfg.ReplaceTunnels(map[string]*Tunnel{"gre0": {Mode: "sit", Local: "192.0.2.1"}}); err != nil {
		t.Fatal(err)
	}

	if cfg.Interfaces["gre0"].VRF != "blue" || cfg.Interfaces["gre0"].Mode != "sit" || cfg.Interfaces["lan"] == nil {
		t.Fatalf("legacy tunnel edit lost interface settings: %+v", cfg.Interfaces)
	}

	if err := cfg.ReplaceTunnels(map[string]*Tunnel{"lan": {Mode: "gre"}}); err == nil {
		t.Fatal("legacy tunnel edit must reject an existing non-tunnel interface")
	}
}

func TestBridgeResolvesNewTunnelMember(t *testing.T) {
	cfg := &Config{Interfaces: map[string]*Interface{
		"gre-office": {Type: "tunnel", Mode: "gretap", Local: "192.0.2.1", Remote: "192.0.2.2"},
		"br-office":  {Type: "bridge", Bridge: &Bridge{Members: []string{"gre-office"}}},
	}}
	var errors []string
	add := func(format string, args ...any) { errors = append(errors, fmt.Sprintf(format, args...)) }
	resolveInterfaces(cfg, add)
	if len(errors) != 0 || len(cfg.Interfaces["br-office"].Bridge.MemberDevices) != 1 || cfg.Interfaces["br-office"].Bridge.MemberDevices[0] != "gre-office" {
		t.Fatalf("new tunnel bridge member requires a kernel interface: %v", errors)
	}
}
