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
}

func migrationFrom(v string) *migration {
	for i := range migrations {
		if migrations[i].from == v {
			return &migrations[i]
		}
	}

	return nil
}

func migrateMap(raw map[string]any) (bool, error) {
	ver, _ := raw["version"].(string)
	if ver == "" {
		return false, nil
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

	return changed, nil
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
