package configtest

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestValidationSetAndNewErrors(t *testing.T) {
	cfg := &config.Config{Hostname: "r1"}
	before := config.ValidationSet(cfg)
	if len(before) != 0 {
		t.Fatalf("base config should be valid, got %v", before)
	}

	cfg.Interfaces = map[string]*config.Interface{"eth0": {}}
	if added := config.NewValidationErrors(before, cfg); len(added) == 0 {
		t.Fatal("expected a new validation error for interface missing select")
	}

	cfg.Interfaces = map[string]*config.Interface{"eth0": {Select: "name=eth0"}}
	if added := config.NewValidationErrors(before, cfg); len(added) != 0 {
		t.Fatalf("valid interface should not error: %v", added)
	}
}

func TestNewValidationErrorsIgnoresPreExisting(t *testing.T) {
	cfg := &config.Config{
		Hostname:   "r1",
		Interfaces: map[string]*config.Interface{"bad": {}},
	}
	before := config.ValidationSet(cfg)
	if len(before) == 0 {
		t.Fatal("expected a pre-existing validation error")
	}

	cfg.Hostname = "r2"
	if added := config.NewValidationErrors(before, cfg); len(added) != 0 {
		t.Fatalf("unrelated change blocked by pre-existing error: %v", added)
	}
}

func TestVXLANValidation(t *testing.T) {
	learning := false
	cfg := &config.Config{
		Hostname: "leaf-1",
		Interfaces: map[string]*config.Interface{
			"vxlan100": {Type: "vxlan", VXLAN: &config.VXLAN{VNI: 100, Local: "192.0.2.1", Port: 4789, Learning: &learning}},
		},
	}

	if errs := config.Validate(cfg, false); len(errs) != 0 {
		t.Fatalf("valid vxlan config: %v", errs)
	}

	cfg.Interfaces["vxlan100"].VXLAN.VNI = 16777216
	if errs := config.Validate(cfg, false); len(errs) == 0 {
		t.Fatal("expected an invalid VNI error")
	}
}

func TestEVPNOptionsRequireEVPNFamily(t *testing.T) {
	cfg := &config.Config{
		Hostname: "leaf-1",
		Routing: &config.Routing{BGP: &config.BGP{
			ASN:      65001,
			RouterID: "192.0.2.1",
			AddressFamilies: map[string]*config.BGPAddressFamily{
				"ipv4-unicast": {AdvertiseAllVNI: true},
			},
		}},
	}

	if errs := config.Validate(cfg, false); len(errs) == 0 {
		t.Fatal("expected EVPN options on ipv4-unicast to fail validation")
	}
}

func TestResolveVXLANBridgeMember(t *testing.T) {
	cfg := &config.Config{
		Interfaces: map[string]*config.Interface{
			"vxlan100": {Type: "vxlan", VXLAN: &config.VXLAN{VNI: 100}},
			"br100":    {Type: "bridge", Bridge: &config.Bridge{Members: []string{"vxlan100"}}},
		},
	}

	config.ResolveInterfaces(cfg)
	got := cfg.Interfaces["br100"].Bridge.MemberDevices
	if len(got) != 1 || got[0] != "vxlan100" {
		t.Fatalf("resolved bridge members = %v, want [vxlan100]", got)
	}
}
