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
		resolveMembers := func() {
			if iface.Bridge == nil {
				return
			}

			for _, sel := range iface.Bridge.Members {
				dev, err := resolveSelector(sel)
				if err != nil {
					add("interfaces.%s.bridge.members: %v", name, err)
					continue
				}

				iface.Bridge.MemberDevices = append(iface.Bridge.MemberDevices, dev)
			}
		}

		switch iface.Type {
		case "dummy", "bridge":
			iface.Device = name
			resolveMembers()
		default:
			if iface.Bridge != nil {
				iface.Device = iface.Select
				resolveMembers()
			} else {
				dev, err := resolveSelector(iface.Select)
				if err != nil {
					add("interfaces.%s: %v", name, err)
					continue
				}

				iface.Device = dev
			}
		}

		for _, vlan := range iface.VLANs {
			vlan.Device = fmt.Sprintf("%s.%d", iface.Device, vlan.ID)
		}

		for _, v := range iface.VRRP {
			for j, sel := range v.TrackInterfaces {
				if iface2, ok := cfg.Interfaces[sel]; ok {
					v.TrackInterfaces[j] = iface2.Device
				} else {
					dev, err := resolveSelector(sel)
					if err != nil {
						add("interfaces.%s.vrrp[*].track_interfaces[%d]: %v", name, j, err)
						continue
					}

					v.TrackInterfaces[j] = dev
				}
			}
		}
	}
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
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join("/sys/class/net", e.Name(), "address"))
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(data)) == mac {
			return e.Name(), nil
		}
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
