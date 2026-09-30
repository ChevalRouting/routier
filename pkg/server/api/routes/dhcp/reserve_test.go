package dhcp

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func dnsServerConfig(zones ...config.DNSZone) *config.Config {
	return &config.Config{
		DNS: &config.DNS{
			Server: &config.DNSServer{Zones: zones},
		},
	}
}

func findRecord(recs []config.DNSRecord, name, rtype string) (config.DNSRecord, bool) {
	for _, r := range recs {
		if r.Name == name && r.Type == rtype {
			return r, true
		}
	}

	return config.DNSRecord{}, false
}

func TestAddReservationDNSForwardAndPTR(t *testing.T) {
	cfg := dnsServerConfig(
		config.DNSZone{Name: "lan.example.com"},
		config.DNSZone{Name: "0.0.10.in-addr.arpa"},
	)

	fqdn, err := addReservationDNS(cfg, "lan.example.com", "printer", "10.0.0.5")
	if err != nil {
		t.Fatalf("addReservationDNS: %v", err)
	}

	if fqdn != "printer.lan.example.com." {
		t.Fatalf("fqdn = %q", fqdn)
	}

	fwd, ok := findRecord(cfg.DNS.Server.Zones[0].Records, "printer", "A")
	if !ok {
		t.Fatalf("forward A record missing: %+v", cfg.DNS.Server.Zones[0].Records)
	}

	if fwd.Value != "10.0.0.5" {
		t.Fatalf("forward value = %q", fwd.Value)
	}

	ptr, ok := findRecord(cfg.DNS.Server.Zones[1].Records, "5", "PTR")
	if !ok {
		t.Fatalf("PTR record missing: %+v", cfg.DNS.Server.Zones[1].Records)
	}

	if ptr.Value != "printer.lan.example.com." {
		t.Fatalf("PTR value = %q", ptr.Value)
	}
}

func TestAddReservationDNSNoReverseZone(t *testing.T) {
	cfg := dnsServerConfig(config.DNSZone{Name: "lan.example.com"})

	if _, err := addReservationDNS(cfg, "lan.example.com", "nas", "10.0.0.9"); err != nil {
		t.Fatalf("addReservationDNS: %v", err)
	}

	if len(cfg.DNS.Server.Zones[0].Records) != 1 {
		t.Fatalf("expected only the forward record, got %+v", cfg.DNS.Server.Zones[0].Records)
	}
}

func TestAddReservationDNSIPv6(t *testing.T) {
	cfg := dnsServerConfig(config.DNSZone{Name: "lan.example.com"})

	if _, err := addReservationDNS(cfg, "lan.example.com", "host", "2001:db8::1"); err != nil {
		t.Fatalf("addReservationDNS: %v", err)
	}

	if _, ok := findRecord(cfg.DNS.Server.Zones[0].Records, "host", "AAAA"); !ok {
		t.Fatalf("AAAA record missing: %+v", cfg.DNS.Server.Zones[0].Records)
	}
}

func TestAddReservationDNSRelativizesFQDN(t *testing.T) {
	cfg := dnsServerConfig(config.DNSZone{Name: "lan.example.com"})

	if _, err := addReservationDNS(cfg, "lan.example.com", "host.lan.example.com.", "10.0.0.7"); err != nil {
		t.Fatalf("addReservationDNS: %v", err)
	}

	if _, ok := findRecord(cfg.DNS.Server.Zones[0].Records, "host", "A"); !ok {
		t.Fatalf("expected relativized label host: %+v", cfg.DNS.Server.Zones[0].Records)
	}
}

func TestAddReservationDNSUnknownZone(t *testing.T) {
	cfg := dnsServerConfig(config.DNSZone{Name: "lan.example.com"})

	if _, err := addReservationDNS(cfg, "other.example.com", "host", "10.0.0.7"); err == nil {
		t.Fatal("expected an error for an unknown zone")
	}
}

func TestAddReservationDNSSkipsSecondaryZone(t *testing.T) {
	cfg := dnsServerConfig(config.DNSZone{Name: "lan.example.com", Primaries: []string{"10.0.0.1"}})

	if _, err := addReservationDNS(cfg, "lan.example.com", "host", "10.0.0.7"); err == nil {
		t.Fatal("expected an error: a secondary zone is not authoritative")
	}
}
