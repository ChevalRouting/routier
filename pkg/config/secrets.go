package config

import (
	"crypto/rand"
	"encoding/base64"
)

const DefaultDDNSAlgorithm = "hmac-sha256"

func EnsureDDNSKey(cfg *Config) bool {
	if cfg == nil || cfg.DHCP == nil || cfg.DHCP.DDNS == nil || !cfg.DHCP.DDNS.Enabled {
		return false
	}

	d := cfg.DHCP.DDNS
	changed := false

	if d.Algorithm == "" {
		d.Algorithm = DefaultDDNSAlgorithm
		changed = true
	}

	if d.Key == "" {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return changed
		}

		d.Key = base64.StdEncoding.EncodeToString(secret)
		changed = true
	}

	return changed
}
