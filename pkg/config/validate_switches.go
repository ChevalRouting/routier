package config

import (
	"regexp"
	"slices"
	"sort"

	"github.com/ChevalRouting/routier/pkg/switches/model"
	"github.com/ChevalRouting/routier/pkg/types"
)

var switchNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)

var switchBondModes = []string{"802.3ad", "active-backup", "balance-xor", "balance-rr", "broadcast", "balance-tlb", "balance-alb"}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

func validateSwitches(cfg *Config, add addfunc) {
	if len(cfg.Switches) == 0 {
		return
	}

	catalog, err := model.Default()
	if err != nil {
		add("switches: switch models: %v", err)
		return
	}

	for _, name := range sortedKeys(cfg.Switches) {
		sw := cfg.Switches[name]
		if sw == nil {
			add("switches.%s: definition is empty", name)
			continue
		}

		validateSwitch(cfg, catalog, name, sw, add)
	}
}

func validateSwitch(cfg *Config, catalog *model.Catalog, name string, sw *Switch, add addfunc) {
	if !switchNamePattern.MatchString(name) {
		add("switches.%s: name must be lowercase letters, digits, dots, dashes or underscores", name)
	}

	if sw.Host == "" {
		add("switches.%s.host is required", name)
	}

	if sw.Scheme != "" && sw.Scheme != "https" && sw.Scheme != "http" {
		add("switches.%s.scheme must be https or http", name)
	}

	if sw.Port != 0 && (sw.Port < 1 || sw.Port > 65535) {
		add("switches.%s.port must be between 1 and 65535", name)
	}

	if sw.Interface != "" && !isKnownInterface(sw.Interface, cfg.Interfaces) {
		add("switches.%s.interface %q not found in interfaces", name, sw.Interface)
	}

	found, err := catalog.Get(sw.Model)
	if err != nil {
		add("switches.%s.model %q is not a known switch model", name, sw.Model)
		return
	}

	ports := faceplatePorts(found)
	checkPort := func(field, port string) {
		if ports != nil && !slices.Contains(ports, port) {
			add("switches.%s.%s: port %q does not exist on %s", name, field, port, found.Name)
		}
	}

	if sw.Uplink != "" {
		checkPort("uplink", sw.Uplink)
	}

	for _, port := range sortedKeys(sw.Ports) {
		checkPort("ports", port)
		validateSwitchPort(add, "switches."+name+".ports."+port, sw.Ports[port])
	}

	members := map[string]string{}
	for _, bond := range sortedKeys(sw.Bonds) {
		validateSwitchBond(add, name, bond, sw, members, checkPort)
	}
}

func validateSwitchBond(add addfunc, name, bond string, sw *Switch, members map[string]string, checkPort func(string, string)) {
	path := "switches." + name + ".bonds." + bond
	cfg := sw.Bonds[bond]
	if cfg == nil {
		add("%s: definition is empty", path)
		return
	}

	if _, clash := sw.Ports[bond]; clash {
		add("%s: a port uses the same name", path)
	}

	if len(cfg.Members) < 2 {
		add("%s.members needs at least two ports", path)
	}

	if cfg.Mode != "" && !slices.Contains(switchBondModes, cfg.Mode) {
		add("%s.mode %q is not a bond mode", path, cfg.Mode)
	}

	for _, member := range cfg.Members {
		checkPort("bonds."+bond+".members", member)
		if _, configured := sw.Ports[member]; configured {
			add("%s.members: %s is also configured as a port", path, member)
		}

		if other, taken := members[member]; taken {
			add("%s.members: %s already belongs to %s", path, member, other)
		}

		members[member] = bond
	}

	if cfg.Port != nil {
		validateSwitchPort(add, path+".port", cfg.Port)
	}
}

func validateSwitchPort(add addfunc, path string, port *SwitchPortConfig) {
	if port == nil {
		add("%s: definition is empty", path)
		return
	}

	switch port.Mode {
	case "access":
		if !validVLANID(port.VLAN) {
			add("%s.vlan must be between 1 and 4094 for an access port", path)
		}
	case "trunk":
		if port.Native != 0 && !validVLANID(port.Native) {
			add("%s.native must be between 1 and 4094", path)
		}

		for _, vid := range port.Tagged {
			if !validVLANID(vid) {
				add("%s.tagged: %d is not between 1 and 4094", path, vid)
			}
		}
	default:
		add("%s.mode must be access or trunk", path)
	}
}

func validVLANID(vid int) bool {
	return vid >= 1 && vid <= 4094
}

func faceplatePorts(m types.SwitchModel) []string {
	if m.Generic || len(m.Faceplate) == 0 {
		return nil
	}

	var ports []string
	for _, group := range m.Faceplate {
		ports = append(ports, group.Ports...)
	}

	return ports
}
