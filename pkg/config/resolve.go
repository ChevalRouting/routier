package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var selectorRe = regexp.MustCompile(`^([a-zA-Z]+)\[(\d+)\]$`)
var macSelectorRe = regexp.MustCompile(`^mac\(([0-9a-fA-F]{2}(?::[0-9a-fA-F]{2}){5})\)$`)

func ResolveInterfaces(cfg *Config) {
	resolveInterfaces(cfg, func(string, ...any) {})
}

func resolveInterfaces(cfg *Config, add addfunc) {
	for name, iface := range cfg.Interfaces {
		resolveBridgeMembers := func() { resolveInterfacesCallback(cfg, add, name, iface) }

		resolveBondMembers := func() { resolveInterfacesCallback2(cfg, add, name, iface) }

		switch iface.Type {
		case "dummy", "bridge", "vxlan", "bond", "vlan":
			iface.Device = name
			resolveBridgeMembers()
			resolveBondMembers()
		default:
			if iface.Bridge != nil {
				iface.Device = iface.Select
				if !isValidIfname(iface.Device) {
					add("interfaces.%s: select %q is used as the bridge device name and %s", name, iface.Select, ifnameRule)
				}

				resolveBridgeMembers()
			} else {
				dev, err := resolveSelector(iface.Select)
				if err != nil {
					add("interfaces.%s: %v", name, err)
					continue
				}

				iface.Device = dev
			}
		}
	}

	if cfg.HA != nil {
		for i, v := range cfg.HA.VRRP {
			for j, sel := range v.TrackInterfaces {
				if iface, ok := cfg.Interfaces[sel]; ok {
					v.TrackInterfaces[j] = iface.Device
				} else {
					dev, err := resolveSelector(sel)
					if err != nil {
						add("ha.vrrp[%d].track_interfaces[%d]: %v", i, j, err)
						continue
					}

					v.TrackInterfaces[j] = dev
				}
			}
		}
	}

	for _, iface := range cfg.Interfaces {
		if iface.VXLAN == nil || iface.VXLAN.VTEP == "" {
			continue
		}

		if vtep, ok := cfg.Interfaces[iface.VXLAN.VTEP]; ok {
			iface.VXLAN.VTEP = vtep.Device
		}
	}
}

func resolveMemberDevice(cfg *Config, sel string) (string, error) {
	if member, ok := cfg.Interfaces[sel]; ok {
		switch member.Type {
		case "dummy", "bridge", "vxlan", "bond":
			return sel, nil
		default:
			return resolveSelector(member.Select)
		}
	}

	return resolveSelector(sel)
}

func resolveSelector(sel string) (string, error) {
	if m := macSelectorRe.FindStringSubmatch(sel); m != nil {
		return resolveByMAC(strings.ToLower(m[1]))
	}

	m := selectorRe.FindStringSubmatch(sel)
	if m == nil {
		return sel, checkIfaceExists(sel)
	}

	prefix := m[1]
	idx, _ := strconv.Atoi(m[2])
	devs, err := listNetDevices(prefix)
	if err != nil {
		return "", err
	}

	if idx >= len(devs) {
		return "", fmt.Errorf("%s: found %d %s* interfaces, need index %d", sel, len(devs), prefix, idx)
	}

	return devs[idx], nil
}

func resolveByMAC(mac string) (string, error) {
	return ResolveMACAt("/sys/class/net", mac)
}

func ResolveMACAt(root, mac string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}

	var physical, virtual []string
	for _, e := range entries {
		base := filepath.Join(root, e.Name())
		data, err := os.ReadFile(filepath.Join(base, "bonding_slave", "perm_hwaddr"))
		if err != nil {
			data, err = os.ReadFile(filepath.Join(base, "address"))
		}

		if err != nil || !strings.EqualFold(strings.TrimSpace(string(data)), mac) {
			continue
		}

		if _, err := os.Stat(filepath.Join(base, "device")); err == nil {
			physical = append(physical, e.Name())
		} else {
			virtual = append(virtual, e.Name())
		}
	}

	matches := physical
	if len(matches) == 0 {
		matches = virtual
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("mac(%s): ambiguous interfaces %s; use an interface name", mac, strings.Join(matches, ", "))
	}

	return "", fmt.Errorf("mac(%s): no interface found", mac)
}

func listNetDevices(prefix string) ([]string, error) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil, err
	}

	var devs []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			devs = append(devs, e.Name())
		}
	}

	sort.Strings(devs)
	return devs, nil
}

func checkIfaceExists(name string) error {
	if _, err := os.Stat(filepath.Join("/sys/class/net", name)); err != nil {
		return fmt.Errorf("interface %s not found", name)
	}

	return nil
}

func resolveInterfacesCallback(cfg *Config, add addfunc, name string, iface *Interface) {
	if iface.Bridge == nil {
		return
	}

	iface.Bridge.MemberDevices = nil
	for _, sel := range iface.Bridge.Members {
		dev, err := resolveMemberDevice(cfg, sel)
		if err != nil {
			add("interfaces.%s.bridge.members: %v", name, err)
			continue
		}

		iface.Bridge.MemberDevices = append(iface.Bridge.MemberDevices, dev)
	}
}

func resolveInterfacesCallback2(cfg *Config, add addfunc, name string, iface *Interface) {
	if iface.Bond == nil {
		return
	}

	iface.Bond.MemberDevices = nil
	iface.Bond.PrimaryDevice = ""
	for _, sel := range iface.Bond.Members {
		dev, err := resolveMemberDevice(cfg, sel)
		if err != nil {
			add("interfaces.%s.bond.members: %v", name, err)
			continue
		}

		iface.Bond.MemberDevices = append(iface.Bond.MemberDevices, dev)
	}

	if iface.Bond.Primary != "" {
		dev, err := resolveMemberDevice(cfg, iface.Bond.Primary)
		if err != nil {
			add("interfaces.%s.bond.primary: %v", name, err)
		} else {
			iface.Bond.PrimaryDevice = dev
		}
	}
}
