package config

import "testing"

func TestDDNSAllowsExistingForwardZone(t *testing.T) {
	cfg := &Config{
		DNS:  &DNS{Server: &DNSServer{Enabled: true, Zones: []DNSZone{{Name: "lan.example.com."}}}},
		DHCP: &DHCP{DDNS: &DHCPDDNS{Enabled: true, Domain: "lan.example.com"}},
	}
	validateDDNS(cfg, func(format string, args ...interface{}) { t.Errorf(format, args...) })
}
