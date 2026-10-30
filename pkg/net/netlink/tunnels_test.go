//go:build linux

package netlink

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestVPNStageReturnsTunnelCreationErrors(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{
		"test-invalid": {Type: "tunnel", Device: "test-invalid", Mode: "invalid"},
	}}
	err := ReconcileLinks(cfg, true)
	if err == nil || !strings.Contains(err.Error(), "test-invalid") {
		t.Fatalf("tunnel failure must stop link reconciliation, got %v", err)
	}
}

func TestTypedTunnelDesiredLinkRetainsVRF(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{
		"gre-office": {Type: "tunnel", Device: "gre-office", Mode: "gre", Local: "192.0.2.1", Remote: "192.0.2.2", MTU: 1400, VRF: "blue"},
	}}
	spec := buildDesiredLinks(cfg)["gre-office"]
	if spec.kind != kindTunnel || spec.vrf != "blue" || spec.mtu != 1400 || spec.tunnel.Remote != "192.0.2.2" {
		t.Fatalf("typed tunnel lost link settings: %+v", spec)
	}
}
