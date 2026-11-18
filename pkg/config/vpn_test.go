package config

import (
	"fmt"
	"strings"
	"testing"
)

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
