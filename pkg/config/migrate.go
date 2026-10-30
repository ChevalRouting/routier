package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type migration struct {
	from  string
	to    string
	apply func(map[string]any) error
}

var migrations = []migration{
	{from: "v1.0.0", to: "v2.0.0", apply: migrateNftablesRulesToBlock},
	{from: "v2.0.0", to: "v3.0.0", apply: migrateDropLegacyHA},
	{from: "v3.0.0", to: "v4.0.0", apply: migrateVPN},
	{from: "v4.0.0", to: "v5.0.0", apply: migrateTunnelInterfaces},
}

func migrationFrom(v string) *migration {
	for i := range migrations {
		if migrations[i].from == v {
			return &migrations[i]
		}
	}

	return nil
}

func dropRetiredFields(raw map[string]any) bool {
	ha, ok := raw["ha"].(map[string]any)
	if !ok {
		return false
	}

	instances, _ := ha["vrrp"].([]any)
	dropped := false
	for _, item := range instances {
		inst, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if _, ok := inst["switchover"]; ok {
			delete(inst, "switchover")
			dropped = true
		}
	}

	return dropped
}

func migrateMap(raw map[string]any) (bool, error) {
	retired := dropRetiredFields(raw)
	ver, _ := raw["version"].(string)
	if ver == "" {
		normalized, err := normalizeNetworkSections(raw)
		return retired || normalized, err
	}

	changed := false
	for ver != CurrentVersion {
		m := migrationFrom(ver)
		if m == nil {
			return changed, fmt.Errorf("no migration path from config version %q toward %q", ver, CurrentVersion)
		}

		if err := m.apply(raw); err != nil {
			return changed, fmt.Errorf("migrate %s -> %s: %w", m.from, m.to, err)
		}

		raw["version"] = m.to
		ver = m.to
		changed = true
	}

	normalized, err := normalizeNetworkSections(raw)
	return retired || changed || normalized, err
}

func MigrateBytes(data []byte) (out []byte, changed bool, err error) {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, false, err
	}

	if raw == nil {
		return data, false, nil
	}

	changed, err = migrateMap(raw)
	if err != nil {
		return nil, false, err
	}

	if !changed {
		return data, false, nil
	}

	out, err = yaml.Marshal(raw)
	if err != nil {
		return nil, false, err
	}

	return out, true, nil
}

func migrateDropLegacyHA(raw map[string]any) error {
	delete(raw, "mirrors")
	delete(raw, "ha_keys")
	return nil
}

func migrateNftablesRulesToBlock(raw map[string]any) error {
	nft, ok := raw["nftables"].(map[string]any)
	if !ok {
		return nil
	}

	chains, ok := nft["chains"].(map[string]any)
	if !ok {
		return nil
	}

	for _, cv := range chains {
		ch, ok := cv.(map[string]any)
		if !ok {
			continue
		}

		seq, ok := ch["rules"].([]any)
		if !ok {
			continue
		}

		lines := make([]string, len(seq))
		for i, it := range seq {
			lines[i] = fmt.Sprint(it)
		}

		ch["rules"] = strings.Join(lines, "\n")
	}

	return nil
}

func migrateVPN(raw map[string]any) error {
	_, err := migrateVPNSections(raw)
	return err
}

func migrateVPNSections(raw map[string]any) (bool, error) {
	changed := false
	for _, section := range []string{"wireguard", "tunnels"} {
		value, exists := raw[section]
		if !exists {
			continue
		}

		vpn, ok := raw["vpn"].(map[string]any)
		if !ok {
			if raw["vpn"] != nil {
				return false, fmt.Errorf("vpn must be a mapping")
			}

			vpn = map[string]any{}
			raw["vpn"] = vpn
		}

		if _, exists := vpn[section]; exists {
			return false, fmt.Errorf("both %s and vpn.%s are configured", section, section)
		}

		vpn[section] = value
		delete(raw, section)
		changed = true
	}

	return changed, nil
}

func migrateTunnelInterfaces(raw map[string]any) error {
	_, err := normalizeTunnelInterfaces(raw)
	return err
}

func normalizeTunnelInterfaces(raw map[string]any) (bool, error) {
	vpn, _ := raw["vpn"].(map[string]any)
	tunnels, exists := vpn["tunnels"]
	if !exists {
		return false, nil
	}

	definitions, ok := tunnels.(map[string]any)
	if !ok {
		return false, fmt.Errorf("vpn.tunnels must be a mapping")
	}

	interfaces, ok := raw["interfaces"].(map[string]any)
	if !ok {
		if raw["interfaces"] != nil {
			return false, fmt.Errorf("interfaces must be a mapping")
		}

		interfaces = map[string]any{}
		raw["interfaces"] = interfaces
	}

	for name, value := range definitions {
		if _, exists := interfaces[name]; exists {
			return false, fmt.Errorf("both interfaces.%s and vpn.tunnels.%s are configured", name, name)
		}

		tunnel, ok := value.(map[string]any)
		if !ok {
			return false, fmt.Errorf("vpn.tunnels.%s must be a mapping", name)
		}

		tunnel["type"] = "tunnel"
		interfaces[name] = tunnel
	}

	delete(vpn, "tunnels")
	if len(vpn) == 0 {
		delete(raw, "vpn")
	}

	return true, nil
}

func normalizeNetworkSections(raw map[string]any) (bool, error) {
	changed, err := migrateVPNSections(raw)
	if err != nil {
		return false, err
	}

	tunnelsChanged, err := normalizeTunnelInterfaces(raw)
	return changed || tunnelsChanged, err
}
