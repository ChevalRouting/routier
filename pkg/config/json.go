package config

import "encoding/json"

type configJSON Config

func (cfg *Config) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	version, _ := raw["version"].(string)
	if migrationFrom(version) != nil {
		if _, err := migrateMap(raw); err != nil {
			return err
		}
	} else if _, err := normalizeNetworkSections(raw); err != nil {
		return err
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	var decoded configJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*cfg = Config(decoded)
	return nil
}
