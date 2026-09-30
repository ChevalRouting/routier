package config

import "testing"

func TestEnsureDDNSKeyGeneratesAndIsIdempotent(t *testing.T) {
	cfg := &Config{DHCP: &DHCP{Enabled: true, DDNS: &DHCPDDNS{Enabled: true, Domain: "lan.example.com"}}}

	if !EnsureDDNSKey(cfg) {
		t.Fatal("EnsureDDNSKey first call = false; want true (key generated)")
	}

	key := cfg.DHCP.DDNS.Key
	if key == "" {
		t.Fatal("EnsureDDNSKey did not set a key")
	}

	if cfg.DHCP.DDNS.Algorithm != DefaultDDNSAlgorithm {
		t.Errorf("algorithm = %q; want %q", cfg.DHCP.DDNS.Algorithm, DefaultDDNSAlgorithm)
	}

	if EnsureDDNSKey(cfg) {
		t.Error("EnsureDDNSKey second call = true; want false (no change)")
	}

	if cfg.DHCP.DDNS.Key != key {
		t.Error("EnsureDDNSKey second call changed the key")
	}
}

func TestEnsureDDNSKeyDisabled(t *testing.T) {
	if EnsureDDNSKey(&Config{}) {
		t.Error("EnsureDDNSKey with no DHCP = true; want false")
	}

	cfg := &Config{DHCP: &DHCP{DDNS: &DHCPDDNS{Enabled: false}}}
	if EnsureDDNSKey(cfg) {
		t.Error("EnsureDDNSKey with DDNS disabled = true; want false")
	}
}
