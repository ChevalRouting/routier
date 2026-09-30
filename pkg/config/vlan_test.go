package config

import (
	"strings"
	"testing"
)

func hasErr(errs []error, substr string) bool {
	for _, err := range errs {
		if strings.Contains(err.Error(), substr) {
			return true
		}
	}

	return false
}

func TestResolveVLANDeviceIsOwnName(t *testing.T) {
	cfg := &Config{
		Hostname: "r",
		Interfaces: map[string]*Interface{
			"lan":     {Type: "dummy"},
			"servers": {Type: "vlan", Select: "lan", VLAN: &VLAN{ID: 100}, Addresses: []string{"10.0.0.1/24"}},
		},
	}

	ResolveInterfaces(cfg)

	if got := cfg.Interfaces["servers"].Device; got != "servers" {
		t.Fatalf("vlan device = %q, want %q", got, "servers")
	}

	if got := cfg.Interfaces["lan"].Device; got != "lan" {
		t.Fatalf("parent device = %q, want %q", got, "lan")
	}
}

func TestValidateVLANValid(t *testing.T) {
	cfg := &Config{
		Hostname: "r",
		Interfaces: map[string]*Interface{
			"lan":     {Type: "dummy"},
			"servers": {Type: "vlan", Select: "lan", VLAN: &VLAN{ID: 100}, Addresses: []string{"10.0.0.1/24"}},
		},
	}

	if errs := Validate(cfg, false); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateVLANRequiresID(t *testing.T) {
	cfg := &Config{
		Hostname: "r",
		Interfaces: map[string]*Interface{
			"lan":     {Type: "dummy"},
			"servers": {Type: "vlan", Select: "lan", VLAN: &VLAN{}},
		},
	}

	if errs := Validate(cfg, false); !hasErr(errs, "vlan.id: id must be 1-4094") {
		t.Fatalf("expected id range error, got: %v", errs)
	}
}

func TestValidateVLANIDOutOfRange(t *testing.T) {
	cfg := &Config{
		Hostname: "r",
		Interfaces: map[string]*Interface{
			"lan":     {Type: "dummy"},
			"servers": {Type: "vlan", Select: "lan", VLAN: &VLAN{ID: 5000}},
		},
	}

	if errs := Validate(cfg, false); !hasErr(errs, "vlan.id: id must be 1-4094") {
		t.Fatalf("expected id range error, got: %v", errs)
	}
}

func TestValidateVLANParentMustExist(t *testing.T) {
	cfg := &Config{
		Hostname: "r",
		Interfaces: map[string]*Interface{
			"servers": {Type: "vlan", Select: "missing", VLAN: &VLAN{ID: 100}},
		},
	}

	if errs := Validate(cfg, false); !hasErr(errs, `parent "missing" not found`) {
		t.Fatalf("expected parent-not-found error, got: %v", errs)
	}
}

func TestValidateVLANSettingsRequireType(t *testing.T) {
	cfg := &Config{
		Hostname: "r",
		Interfaces: map[string]*Interface{
			"lan": {Select: "eth0", VLAN: &VLAN{ID: 100}},
		},
	}

	if errs := Validate(cfg, false); !hasErr(errs, "vlan settings require type vlan") {
		t.Fatalf("expected settings-require-type error, got: %v", errs)
	}
}

func TestValidateVLANTypeRequiresSettings(t *testing.T) {
	cfg := &Config{
		Hostname: "r",
		Interfaces: map[string]*Interface{
			"lan":     {Type: "dummy"},
			"servers": {Type: "vlan", Select: "lan"},
		},
	}

	if errs := Validate(cfg, false); !hasErr(errs, "vlan settings are required") {
		t.Fatalf("expected settings-required error, got: %v", errs)
	}
}

func TestValidateVLANDeviceName(t *testing.T) {
	cfg := &Config{
		Hostname: "r",
		Interfaces: map[string]*Interface{
			"lan":                       {Type: "dummy"},
			"this-name-is-way-too-long": {Type: "vlan", Select: "lan", VLAN: &VLAN{ID: 100}},
		},
	}

	if errs := Validate(cfg, false); !hasErr(errs, ifnameRule) {
		t.Fatalf("expected ifname error, got: %v", errs)
	}
}
