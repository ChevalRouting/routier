package render

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestReverseZoneName(t *testing.T) {
	cases := []struct {
		cidr string
		want string
		ok   bool
	}{
		{"192.168.10.0/24", "10.168.192.in-addr.arpa", true},
		{"10.0.0.0/8", "10.in-addr.arpa", true},
		{"172.16.0.0/16", "16.172.in-addr.arpa", true},
		{"2001:db8:10::/64", "0.0.0.0.0.1.0.0.8.b.d.0.1.0.0.2.ip6.arpa", true},
		{"192.168.10.0/25", "", false},
		{"2001:db8::/63", "", false},
		{"bogus", "", false},
	}

	for _, c := range cases {
		got, ok := reverseZoneName(c.cidr)
		if ok != c.ok || got != c.want {
			t.Errorf("reverseZoneName(%q) = %q, %v; want %q, %v", c.cidr, got, ok, c.want, c.ok)
		}
	}
}

func ddnsCfg() *config.Config {
	return &config.Config{
		Hostname: "gw",
		DNS:      &config.DNS{Server: &config.DNSServer{Enabled: true}},
		DHCP: &config.DHCP{
			Enabled:  true,
			Subnets4: []config.KeaSubnet{{Subnet: "192.168.10.0/24", Pools: []string{"192.168.10.100-192.168.10.200"}}},
			DDNS: &config.DHCPDDNS{
				Enabled:   true,
				Domain:    "lan.example.com",
				Key:       "c2VjcmV0Cg==",
				Algorithm: "hmac-sha256",
			},
		},
	}
}

func TestRenderKeaDDNS(t *testing.T) {
	out, err := renderKeaDDNS(ddnsCfg())
	if err != nil {
		t.Fatalf("renderKeaDDNS: %v", err)
	}

	if len(out) != 1 {
		t.Fatalf("renderKeaDDNS outputs = %d; want 1", len(out))
	}

	content := out[0].Content
	for _, want := range []string{
		`"DhcpDdns"`,
		`"port": 53001`,
		`"name": "routier-ddns"`,
		`"algorithm": "HMAC-SHA256"`,
		`"secret": "c2VjcmV0Cg=="`,
		`"lan.example.com."`,
		`"10.168.192.in-addr.arpa."`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("kea-dhcp-ddns.conf missing %q\n%s", want, content)
		}
	}
}

func TestRenderKeaDDNSInactiveWithoutBind(t *testing.T) {
	cfg := ddnsCfg()
	cfg.DNS.Server.Enabled = false
	if out, err := renderKeaDDNS(cfg); err != nil || out != nil {
		t.Fatalf("renderKeaDDNS without bind = %v, %v; want nil, nil", out, err)
	}
}

func TestRenderKeaDHCP4DDNSParams(t *testing.T) {
	out, err := renderKea(ddnsCfg())
	if err != nil {
		t.Fatalf("renderKea: %v", err)
	}

	var content string
	for _, o := range out {
		if o.Name == "kea/kea-dhcp4.conf" {
			content = o.Content
		}
	}

	for _, want := range []string{
		`"dhcp-ddns"`,
		`"enable-updates": true`,
		`"ddns-send-updates": true`,
		`"ddns-qualifying-suffix": "lan.example.com"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("kea-dhcp4.conf missing %q\n%s", want, content)
		}
	}
}

func TestRenderNamedConfDDNS(t *testing.T) {
	out, err := All(ddnsCfg())
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	var named string
	for _, o := range out {
		if o.Name == NamedConfName {
			named = o.Content
		}
	}

	if named == "" {
		t.Fatal("named.conf not rendered")
	}

	for _, want := range []string{
		`key "routier-ddns"`,
		`algorithm hmac-sha256;`,
		`zone "lan.example.com." IN {`,
		`allow-update { key "routier-ddns"; };`,
		`allow-transfer { key "routier-ddns"; };`,
		`zone "10.168.192.in-addr.arpa." IN {`,
	} {
		if !strings.Contains(named, want) {
			t.Errorf("named.conf missing %q\n%s", want, named)
		}
	}
}

func TestDDNSOverlappingForwardZones(t *testing.T) {
	cfg := ddnsCfg()
	cfg.DHCP.Subnets4 = append(cfg.DHCP.Subnets4, config.KeaSubnet{
		Subnet:     "192.168.20.0/24",
		Pools:      []string{"192.168.20.100-192.168.20.200"},
		DDNSDomain: "iot.example.com.",
	})

	zones := ddnsForwardZones(cfg)
	want := []string{"lan.example.com", "iot.example.com"}
	if len(zones) != len(want) || zones[0] != want[0] || zones[1] != want[1] {
		t.Fatalf("ddnsForwardZones = %v; want %v", zones, want)
	}

	out, err := All(cfg)
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	var named, keaddns string
	for _, o := range out {
		switch o.Name {
		case NamedConfName:
			named = o.Content
		case "kea/kea-dhcp-ddns.conf":
			keaddns = o.Content
		}
	}

	for _, want := range []string{`zone "lan.example.com." IN {`, `zone "iot.example.com." IN {`} {
		if !strings.Contains(named, want) {
			t.Errorf("named.conf missing %q", want)
		}
	}

	for _, want := range []string{`"name": "lan.example.com."`, `"name": "iot.example.com."`} {
		if !strings.Contains(keaddns, want) {
			t.Errorf("kea-dhcp-ddns.conf missing forward domain %q\n%s", want, keaddns)
		}
	}
}

func TestDDNSForwardZoneDedup(t *testing.T) {
	cfg := ddnsCfg()
	cfg.DHCP.Subnets4[0].DDNSDomain = "lan.example.com"

	if zones := ddnsForwardZones(cfg); len(zones) != 1 || zones[0] != "lan.example.com" {
		t.Fatalf("ddnsForwardZones = %v; want single lan.example.com", zones)
	}
}

func TestRenderKeaDDNSReverseDisabled(t *testing.T) {
	cfg := ddnsCfg()
	no := false
	cfg.DHCP.DDNS.Reverse = &no

	if got := ddnsReverseZoneNames(cfg); got != nil {
		t.Errorf("ddnsReverseZoneNames with reverse disabled = %v; want nil", got)
	}
}

func TestDDNSReusesExistingZones(t *testing.T) {
	cfg := ddnsCfg()
	for _, name := range []string{"LAN.example.com.", "10.168.192.in-addr.arpa."} {
		cfg.DNS.Server.Zones = append(cfg.DNS.Server.Zones, config.DNSZone{
			Name: name, Nameservers: []string{"ns.example.com."},
			Records: []config.DNSRecord{{Name: "existing", Type: "TXT", Value: "preserved"}},
		})
	}
	out, err := All(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range out {
		if strings.HasPrefix(o.Dest, NamedZoneDir+"/") {
			t.Fatalf("dynamic zone would be overwritten: %s", o.Dest)
		}
		if o.Name == NamedConfName {
			if got := strings.Count(o.Content, "allow-update { key"); got != 2 {
				t.Fatalf("got %d DDNS declarations, want 2", got)
			}
			if got := strings.Count(o.Content, "type primary;"); got != 2 {
				t.Fatalf("got %d primary zones, want 2", got)
			}
		}
	}
	for _, name := range DDNSZoneNames(cfg) {
		if content := DDNSBootstrapZoneFile(cfg, name); !strings.Contains(content, "preserved") || !strings.Contains(content, "ns.example.com.") {
			t.Fatalf("bootstrap lost existing zone data: %s", content)
		}
	}
	cfg.DNS.Server.Zones[0].Primaries = []string{"192.0.2.1"}
	if _, err := All(cfg); err == nil {
		t.Fatal("DDNS must reject a secondary zone")
	}
}

func TestDDNSForwardParentZone(t *testing.T) {
	cfg := ddnsCfg()
	cfg.DHCP.DDNS.Domain = "ans.mvinc.fr"
	cfg.DHCP.Subnets4[0].DDNSDomain = "iot.mvinc.fr"
	cfg.DNS.Server.Zones = []config.DNSZone{{Name: "MVINC.FR.", Nameservers: []string{"ns.mvinc.fr."}, Records: []config.DNSRecord{{Name: "ns", Type: "A", Value: "192.0.2.1"}}}}
	zones := ddnsForwardZones(cfg)
	if len(zones) != 1 || zones[0] != "mvinc.fr" {
		t.Fatalf("update zones = %v", zones)
	}
	out, err := All(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range out {
		switch o.Name {
		case NamedConfName:
			if strings.Count(o.Content, `zone "mvinc.fr." IN {`) != 1 || strings.Contains(o.Content, `zone "ans.mvinc.fr.`) || strings.Contains(o.Content, `zone "iot.mvinc.fr.`) {
				t.Fatalf("unexpected zone declarations: %s", o.Content)
			}
			if !strings.Contains(o.Content, "file \"/etc/bind/zones/mvinc.fr.zone\";\n    allow-update { key \"routier-ddns\"; };") {
				t.Fatal("parent zone missing update permission")
			}
		case "kea/kea-dhcp-ddns.conf":
			if !strings.Contains(o.Content, `"name": "mvinc.fr."`) || strings.Contains(o.Content, `"name": "ans.mvinc.fr."`) {
				t.Fatalf("wrong D2 update target: %s", o.Content)
			}
		case "kea/kea-dhcp4.conf":
			for _, suffix := range []string{"ans.mvinc.fr", "iot.mvinc.fr"} {
				if !strings.Contains(o.Content, `"ddns-qualifying-suffix": "`+suffix+`"`) {
					t.Fatalf("lost hostname suffix %s", suffix)
				}
			}
		}
		if strings.HasPrefix(o.Dest, NamedZoneDir+"/") {
			t.Fatalf("parent zone would be overwritten: %s", o.Dest)
		}
	}
}

func TestDDNSForwardClosestZone(t *testing.T) {
	for _, tc := range []struct{ domain, want string }{
		{"host.ANS.mvinc.fr.", "ans.mvinc.fr"},
		{"ans.mvinc.fr", "ans.mvinc.fr"},
		{"other.mvinc.fr", "mvinc.fr"},
		{"notmvinc.fr", "notmvinc.fr"},
		{"", ""},
	} {
		t.Run(tc.domain, func(t *testing.T) {
			cfg := ddnsCfg()
			cfg.DNS.Server.Zones = []config.DNSZone{{Name: "ans.mvinc.fr."}, {Name: "mvinc.fr"}}
			if got := ddnsForwardUpdateZone(cfg, tc.domain); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	cfg := ddnsCfg()
	cfg.DHCP.DDNS.Domain = "host.ans.mvinc.fr"
	cfg.DNS.Server.Zones = []config.DNSZone{{Name: "mvinc.fr"}, {Name: "ans.mvinc.fr", Primaries: []string{"192.0.2.1"}}}
	if _, err := All(cfg); err == nil {
		t.Fatal("must reject a secondary parent instead of updating its ancestor")
	}
}
