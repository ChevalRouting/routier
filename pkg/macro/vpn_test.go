package macro

import (
	"reflect"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestVPNDeltaPreservesUnrelatedLiveTunnels(t *testing.T) {
	base := &config.Config{}
	live := &config.Config{Interfaces: map[string]*config.Interface{"gre-live": {Type: "tunnel", Mode: "gre"}}}
	mod := &config.Config{VPN: config.VPNConfig{Wireguard: map[string]*config.Wireguard{"wg-new": {ListenPort: 51820}}}}
	sections := ComputeSections(base, mod)
	if !reflect.DeepEqual(sections, []string{"vpn"}) {
		t.Fatalf("expected canonical VPN section, got %v", sections)
	}

	if conflicts := DetectConflicts(base, live, mod, sections); len(conflicts) != 0 {
		t.Fatalf("unrelated VPN changes must not conflict: %v", conflicts)
	}

	merged := ApplyDelta(base, live, mod, sections)
	if merged.Interfaces["gre-live"] == nil || merged.VPN.Wireguard["wg-new"] == nil {
		t.Fatalf("VPN merge lost unrelated definitions: %+v", merged.VPN)
	}
}

func TestLegacyWireguardMacroSectionUsesVPN(t *testing.T) {
	base := &config.Config{}
	mod := &config.Config{VPN: config.VPNConfig{Wireguard: map[string]*config.Wireguard{"wg-new": {ListenPort: 51820}}}}
	merged := ApplyDelta(base, base, mod, []string{"wireguard"})
	if merged.VPN.Wireguard["wg-new"] == nil {
		t.Fatal("legacy macro section must target nested VPN configuration")
	}
}

func TestLegacyTunnelMacroPreservesOtherInterfaces(t *testing.T) {
	base := &config.Config{}
	live := &config.Config{Interfaces: map[string]*config.Interface{"lan": {Type: "dummy"}}}
	mod := &config.Config{Interfaces: map[string]*config.Interface{"gre-office": {Type: "tunnel", Mode: "gre", Local: "192.0.2.1"}}}
	merged := ApplyDelta(base, live, mod, []string{"tunnels"})
	if merged.Interfaces["lan"] == nil || merged.Interfaces["gre-office"] == nil || merged.Interfaces["gre-office"].Type != "tunnel" {
		t.Fatalf("legacy tunnel macro lost interfaces: %+v", merged.Interfaces)
	}
}
