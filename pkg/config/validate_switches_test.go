package config

import (
	"fmt"
	"strings"
	"testing"
)

func switchConfig(sw *Switch) *Config {
	return &Config{
		Hostname:   "edge-1",
		Interfaces: map[string]*Interface{"lan": {Type: "dummy"}},
		Switches:   map[string]*Switch{"crs310": sw},
	}
}

func switchErrors(sw *Switch) []string {
	var messages []string
	add := func(msg string, args ...any) { messages = append(messages, fmt.Sprintf(msg, args...)) }
	validateSwitches(switchConfig(sw), add)
	return messages
}

func TestValidateSwitchesAcceptsAGoodDefinition(t *testing.T) {
	sw := &Switch{
		Model: "mikrotik-crs310-8g-2s", Host: "10.0.0.2", Scheme: "http", Interface: "lan", Uplink: "sfp-sfpplus1",
		Ports: map[string]*SwitchPortConfig{
			"ether1": {Mode: "access", VLAN: 20},
			"ether2": {Mode: "trunk", Native: 10, Tagged: []int{20, 30}},
		},
		Bonds: map[string]*SwitchBondConfig{
			"bond1": {Members: []string{"ether7", "ether8"}, Mode: "802.3ad", Port: &SwitchPortConfig{Mode: "trunk", Tagged: []int{10}}},
		},
	}
	if errs := switchErrors(sw); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateSwitchesReportsMistakes(t *testing.T) {
	sw := &Switch{
		Model: "mikrotik-crs310-8g-2s", Interface: "wan", Uplink: "sfp-sfpplus9", Port: 70000, Scheme: "ftp",
		Ports: map[string]*SwitchPortConfig{
			"ether1":  {Mode: "access"},
			"ether2":  {Mode: "trunk", Tagged: []int{5000}},
			"ether3":  {Mode: "hybrid"},
			"ether99": {Mode: "access", VLAN: 10},
		},
		Bonds: map[string]*SwitchBondConfig{
			"bond1": {Members: []string{"ether3"}, Mode: "lacp"},
			"bond2": {Members: []string{"ether5", "ether6"}},
			"bond3": {Members: []string{"ether6", "ether7"}},
		},
	}
	errs := strings.Join(switchErrors(sw), "\n")
	want := []string{
		"switches.crs310.host is required",
		"port must be between 1 and 65535",
		"scheme must be https or http",
		`interface "wan" not found`,
		`uplink: port "sfp-sfpplus9" does not exist`,
		"ports.ether1.vlan must be between 1 and 4094",
		"ports.ether2.tagged: 5000",
		"ports.ether3.mode must be access or trunk",
		`port "ether99" does not exist`,
		"bonds.bond1.members needs at least two ports",
		`bonds.bond1.mode "lacp"`,
		"bonds.bond1.members: ether3 is also configured as a port",
		"bonds.bond3.members: ether6 already belongs to bond2",
	}
	for _, fragment := range want {
		if !strings.Contains(errs, fragment) {
			t.Errorf("missing %q in:\n%s", fragment, errs)
		}
	}
}

func TestValidateSwitchesUnknownModelAndName(t *testing.T) {
	cfg := switchConfig(&Switch{Model: "cisco-9300", Host: "10.0.0.3"})
	cfg.Switches["Bad Name"] = cfg.Switches["crs310"]
	var messages []string
	add := func(msg string, args ...any) { messages = append(messages, fmt.Sprintf(msg, args...)) }
	validateSwitches(cfg, add)
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, `model "cisco-9300" is not a known switch model`) || !strings.Contains(joined, "switches.Bad Name: name must be") {
		t.Fatalf("unexpected errors:\n%s", joined)
	}
}

func TestValidateSwitchesGenericModelAcceptsAnyPort(t *testing.T) {
	sw := &Switch{Model: "mikrotik-generic", Host: "10.0.0.4", Ports: map[string]*SwitchPortConfig{"ether24": {Mode: "access", VLAN: 10}}}
	if errs := switchErrors(sw); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}
