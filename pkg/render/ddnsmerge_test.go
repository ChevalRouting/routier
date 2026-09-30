package render

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

const frozenZoneSample = `$ORIGIN .
$TTL 3600	; 1 hour
lan.example.com		IN SOA	ns.lan.example.com. hostmaster.lan.example.com. (
				10         ; serial
				3600       ; refresh
				600        ; retry
				604800     ; expire
				300        ; minimum
				)
			NS	ns.lan.example.com.
$ORIGIN lan.example.com.
ns			A	192.168.10.1
laptop			3600 IN	A	192.168.10.150
laptop			3600 IN	DHCID	AAABxxxxLease==
printer			A	192.168.10.170
printer			DHCID	AAABbbbbLease==
`

func mergeCfg() *config.Config {
	cfg := ddnsCfg()
	cfg.DNS.Server.Zones = []config.DNSZone{{
		Name:        "lan.example.com",
		Nameservers: []string{"ns.lan.example.com"},
		Records: []config.DNSRecord{
			{Name: "ns", Type: "A", Value: "192.168.10.1"},
			{Name: "server", Type: "AAAA", Value: "fd00::1"},
		},
	}}

	return cfg
}

func TestParseZoneFileSerial(t *testing.T) {
	_, serial := parseZoneFile(frozenZoneSample, config.NormalizeDNSName("lan.example.com"))
	if serial != 10 {
		t.Fatalf("serial = %d; want 10", serial)
	}
}

func TestMergeDDNSZonePreservesDynamic(t *testing.T) {
	merged := MergeDDNSZone(mergeCfg(), "lan.example.com", frozenZoneSample)

	mustContain := []string{
		"server", "fd00::1",
		"laptop", "192.168.10.150", "AAABxxxxLease==",
		"printer", "AAABbbbbLease==",
		"( 11 ",
	}
	for _, want := range mustContain {
		if !strings.Contains(merged, want) {
			t.Errorf("merged zone missing %q\n%s", want, merged)
		}
	}

	if strings.Contains(merged, "10         ; serial") {
		t.Errorf("merged zone kept the old serial\n%s", merged)
	}

	if got := strings.Count(merged, "SOA"); got != 1 {
		t.Errorf("merged zone SOA count = %d; want 1\n%s", got, merged)
	}
}

func TestMergeDDNSZoneStaticOwnerWins(t *testing.T) {
	frozen := frozenZoneSample + "server\t\tA\t192.168.10.9\nserver\t\tDHCID\tAAABserverLease==\n"

	merged := MergeDDNSZone(mergeCfg(), "lan.example.com", frozen)

	if strings.Contains(merged, "192.168.10.9") {
		t.Errorf("static owner should override the dynamic record\n%s", merged)
	}

	if !strings.Contains(merged, "fd00::1") {
		t.Errorf("static AAAA missing\n%s", merged)
	}
}

func TestRenderDNSZonesEmitsDDNSStaticShadow(t *testing.T) {
	out, err := renderDNSZones(mergeCfg())
	if err != nil {
		t.Fatalf("renderDNSZones: %v", err)
	}

	var shadow *Output
	for i := range out {
		if out[i].Name == namedZonePfx+ZoneFileName("lan.example.com") {
			shadow = &out[i]
		}
	}

	if shadow == nil {
		t.Fatalf("no shadow output for the DDNS zone\n%+v", out)
	}

	if shadow.Dest != staticShadowPath("lan.example.com") {
		t.Errorf("shadow Dest = %q; want %q", shadow.Dest, staticShadowPath("lan.example.com"))
	}

	if !strings.Contains(shadow.Content, "fd00::1") {
		t.Errorf("shadow content missing static AAAA\n%s", shadow.Content)
	}
}
