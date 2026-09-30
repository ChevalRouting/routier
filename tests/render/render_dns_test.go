package rendertest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/tests/testkit"
)

const dnsTargetConfig = `version: v3.0.0
hostname: cgw-1
interfaces:
  region:
    select: name=eth0
    addresses: ["10.170.32.2/24"]
  workstations:
    select: name=eth1
    addresses: ["10.170.48.2/20"]
  external:
    select: name=eth2
    addresses: ["10.170.33.2/24"]
  dum0:
    type: dummy
    addresses: ["10.255.0.53/32"]
ha:
  vrrp:
    - id: 32
      interface: region
      vips: ["10.170.32.1/24"]
    - id: 48
      interface: workstations
      vips: ["10.170.48.1/20"]
dns:
  nameservers: ["127.0.0.1"]
  search: ["42.school"]
  server:
    enabled: true
    listen:
      - 127.0.0.1
      - iface(region)
      - vips(region)
      - iface(workstations)
      - vips(workstations)
      - iface(external)
      - iface(dum0)
    allow_from:
      - 10.170.32.0/24
      - 10.170.48.0/20
      - 10.170.33.0/24
      - 127.0.0.1/32
      - 10.255.0.53/32
    allow_inbound: [region, workstations, external]
    upstreams: [1.1.1.1]
    forward:
      - domain: 10.in-addr.arpa
        servers: [10.255.0.54]
      - domain: 42.school
        servers: [10.255.0.54]
    cache:
      disabled: true
`

const dnsZoneConfig = `version: v3.0.0
hostname: gw
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.1/24"]
dns:
  server:
    enabled: true
    listen: [iface(lan)]
    allow_from: [10.0.0.0/24]
    allow_inbound: [lan]
    upstreams: [1.1.1.1, 9.9.9.9]
    cache: { size: 64m, max_ttl: 3600, prefetch: true }
    zones:
      - name: home.arpa
        ttl: 300
        nameservers: [ns1.home.arpa.]
        soa:
          primary: ns1.home.arpa.
          email: hostmaster@home.arpa
          refresh: 3600
          retry: 600
          expire: 604800
          minimum: 300
        records:
          - { name: "@",   type: A,     value: 10.0.0.1 }
          - { name: ns1,   type: A,     value: 10.0.0.1 }
          - { name: nas,   type: A,     value: 10.0.0.10 }
          - { name: nas,   type: AAAA,  value: "2001:db8::10" }
          - { name: files, type: CNAME, value: nas.home.arpa. }
          - { name: "@",   type: MX,    value: mail.home.arpa., priority: 10 }
          - { name: "@",   type: TXT,   value: "managed by routier" }
      - name: 0.0.10.in-addr.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "1",  type: PTR, value: gw.home.arpa. }
          - { name: "10", type: PTR, value: nas.home.arpa. }
`

func dnsCfgRender(t *testing.T, cfg *config.Config) map[string]string {
	t.Helper()

	outputs, err := render.All(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	byDest := make(map[string]string, len(outputs))
	for _, o := range outputs {
		byDest[o.Dest] = o.Content
	}

	return byDest
}

func namedConf(t *testing.T, yaml string) string {
	t.Helper()

	conf, ok := renderByDest(t, yaml)[render.NamedConfDest]
	if !ok {
		t.Fatalf("no output rendered for %s", render.NamedConfDest)
	}

	return conf
}

func wantLines(t *testing.T, label, content string, wants ...string) {
	t.Helper()

	for _, w := range wants {
		if !strings.Contains(content, w) {
			t.Errorf("%s missing %q in:\n%s", label, w, content)
		}
	}
}

func TestRenderNamedTarget(t *testing.T) {
	conf := namedConf(t, dnsTargetConfig)

	wantLines(t, "named.conf", conf,
		`include "/etc/bind/rndc.key";`,
		"inet 127.0.0.1 port 953 allow { 127.0.0.1; } keys { \"rndc-key\"; };",
		"acl routier_clients {",
		"10.170.32.0/24;",
		"10.170.48.0/20;",
		"10.170.33.0/24;",
		"127.0.0.1/32;",
		"10.255.0.53/32;",
		"listen-on port 53 { 10.170.32.1; 10.170.32.2; 10.170.33.2; 10.170.48.1; 10.170.48.2; 10.255.0.53; 127.0.0.1; };",
		"listen-on-v6 { none; };",
		"allow-query { routier_clients; };",
		"allow-transfer { none; };",
		"recursion yes;",
		"allow-recursion { routier_clients; };",
		"dnssec-validation no;",
		"max-cache-ttl 0;",
		"max-ncache-ttl 0;",
		`statistics-file "`+render.NamedStats+`";`,
		`file "`+render.NamedLog+`" versions 3 size 10m;`,
	)

	for _, block := range []string{
		"forwarders {\n        1.1.1.1;\n    };\n    forward only;",
		"zone \"10.in-addr.arpa.\" IN {\n    type forward;\n    forward only;\n    forwarders {\n        10.255.0.54;",
		"zone \"42.school.\" IN {\n    type forward;\n    forward only;\n    forwarders {\n        10.255.0.54;",
	} {
		if !strings.Contains(conf, block) {
			t.Errorf("named.conf missing block:\n%s\ngot:\n%s", block, conf)
		}
	}

	if strings.Contains(conf, "max-cache-size") {
		t.Error("cache.disabled must zero the TTLs, not set a cache size")
	}

	if strings.Contains(conf, "dnssec-validation auto") {
		t.Errorf("dnssec validation enabled without dns.server.dnssec:\n%s", conf)
	}
}

func TestRenderNamedSpecialDomains(t *testing.T) {
	conf := namedConf(t, dnsTargetConfig)

	wantLines(t, "named.conf", conf, "validate-except {\n        \"10.in-addr.arpa\";\n        \"42.school\";\n    };")

	if !strings.Contains(conf, `disable-empty-zone "10.in-addr.arpa";`) {
		t.Errorf("10.in-addr.arpa has a BIND empty zone and must be disabled:\n%s", conf)
	}

	if strings.Contains(conf, `disable-empty-zone "42.school";`) {
		t.Errorf("42.school has no BIND empty zone; disabling one is noise:\n%s", conf)
	}

	zoneConf := namedConf(t, dnsZoneConfig)
	for _, name := range []string{"home.arpa", "0.0.10.in-addr.arpa"} {
		wantLines(t, "named.conf", zoneConf,
			`        "`+name+`";`,
			`disable-empty-zone "`+name+`";`,
		)
	}

	if strings.Count(zoneConf, "validate-except {") != 1 {
		t.Errorf("validate-except takes a list; BIND rejects repeats:\n%s", zoneConf)
	}

	signed := &config.Config{
		Hostname:   "gw",
		Interfaces: map[string]*config.Interface{"lan": {Device: "eth0", Addresses: []string{"10.0.0.1/24"}}},
		DNS: &config.DNS{Server: &config.DNSServer{
			Enabled:   true,
			Listen:    []string{"iface(lan)"},
			AllowFrom: []string{"10.0.0.0/24"},
			Forward: []config.DNSForward{
				{Domain: "signed.example", Servers: []string{"10.0.0.53"}, DNSSEC: true},
				{Domain: "internal.example", Servers: []string{"10.0.0.53"}},
			},
			Zones: []config.DNSZone{
				{
					Name:        "signed.zone",
					Nameservers: []string{"ns1.signed.zone."},
					SOA:         &config.DNSSOA{Email: "hostmaster@signed.zone"},
					Records:     []config.DNSRecord{{Name: "@", Type: "A", Value: "10.0.0.1"}},
					DNSSEC:      true,
				},
				{
					Name:        "local.zone",
					Nameservers: []string{"ns1.local.zone."},
					SOA:         &config.DNSSOA{Email: "hostmaster@local.zone"},
					Records:     []config.DNSRecord{{Name: "@", Type: "A", Value: "10.0.0.1"}},
				},
			},
		}},
	}

	conf = dnsCfgRender(t, signed)[render.NamedConfDest]
	for _, name := range []string{"internal.example", "local.zone"} {
		wantLines(t, "named.conf", conf, `        "`+name+`";`)
	}

	for _, name := range []string{"signed.example", "signed.zone"} {
		if strings.Contains(conf, `"`+name+`";`) {
			t.Errorf("dnssec: true entry must still be validated, got validate-except for %q", name)
		}
	}
}

func TestRenderNamedListenResolvesRefs(t *testing.T) {
	conf := namedConf(t, `version: v3.0.0
hostname: gw
interfaces:
  region:
    select: name=eth0
    addresses: ["10.170.32.2/24"]
  guests:
    type: vlan
    select: region
    addresses: ["10.170.40.2/24"]
    vlan:
      id: 40
  wan:
    select: name=eth1
    addresses: [dhcp]
ha:
  vrrp:
    - id: 32
      interface: region
      vips: ["10.170.32.1/24"]
dns:
  server:
    enabled: true
    listen:
      - 127.0.0.1
      - iface(region)
      - vips(region)
      - iface(guests)
      - 10.170.32.2
    allow_from: [10.170.32.0/24]
`)

	want := "listen-on port 53 { 10.170.32.1; 10.170.32.2; 10.170.40.2; 127.0.0.1; };"
	if !strings.Contains(conf, want) {
		t.Errorf("expected sorted, de-duplicated %q in:\n%s", want, conf)
	}
}

func TestRenderNamedZoneFile(t *testing.T) {
	out := renderByDest(t, dnsZoneConfig)

	zone, ok := out[render.NamedZoneDir+"/home.arpa.zone"]
	if !ok {
		t.Fatalf("home.arpa zone file not rendered, got %v", destKeys(out))
	}

	wantLines(t, "home.arpa.zone", zone,
		"$ORIGIN home.arpa.\n",
		"$TTL 300\n",
		"IN SOA   ns1.home.arpa. hostmaster.home.arpa. (",
		"IN NS    ns1.home.arpa.",
		"IN A     10.0.0.1",
		"IN AAAA  2001:db8::10",
		"IN CNAME nas.home.arpa.",
		"IN MX    10 mail.home.arpa.",
		`IN TXT   "managed by routier"`,
	)

	if n := strings.Count(zone, " SOA "); n != 1 {
		t.Errorf("expected exactly one SOA record, got %d:\n%s", n, zone)
	}

	reverse, ok := out[render.NamedZoneDir+"/0.0.10.in-addr.arpa.zone"]
	if !ok {
		t.Fatalf("reverse zone file not rendered, got %v", destKeys(out))
	}

	wantLines(t, "reverse zone", reverse,
		"$ORIGIN 0.0.10.in-addr.arpa.\n",
		"IN PTR gw.home.arpa.",
		"IN PTR nas.home.arpa.",
	)

	conf := out[render.NamedConfDest]
	for _, block := range []string{
		"zone \"home.arpa.\" IN {\n    type primary;\n    file \"" + render.NamedZoneDir + "/home.arpa.zone\";",
		"zone \"0.0.10.in-addr.arpa.\" IN {\n    type primary;\n    file \"" + render.NamedZoneDir + "/0.0.10.in-addr.arpa.zone\";",
	} {
		if !strings.Contains(conf, block) {
			t.Errorf("named.conf missing primary zone block:\n%s\ngot:\n%s", block, conf)
		}
	}
}

func destKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func TestRenderNamedZoneSerialIsDeterministic(t *testing.T) {
	first := renderByDest(t, dnsZoneConfig)[render.NamedZoneDir+"/home.arpa.zone"]
	second := renderByDest(t, dnsZoneConfig)[render.NamedZoneDir+"/home.arpa.zone"]

	if first == "" {
		t.Fatal("home.arpa zone file not rendered")
	}

	if first != second {
		t.Errorf("two renders of the same config differ:\n%s\n---\n%s", first, second)
	}

	changed := renderByDest(t, strings.Replace(dnsZoneConfig, "value: 10.0.0.10", "value: 10.0.0.11", 1))[render.NamedZoneDir+"/home.arpa.zone"]
	if zoneSerialOf(t, changed) == zoneSerialOf(t, first) {
		t.Error("serial did not change after a record changed")
	}

	explicit := renderByDest(t, strings.Replace(dnsZoneConfig, "refresh: 3600", "serial: 2024010101\n          refresh: 3600", 1))[render.NamedZoneDir+"/home.arpa.zone"]
	if got := zoneSerialOf(t, explicit); got != "2024010101" {
		t.Errorf("explicit soa.serial = %q, want 2024010101", got)
	}
}

func zoneSerialOf(t *testing.T, zone string) string {
	t.Helper()

	_, rest, ok := strings.Cut(zone, "( ")
	if !ok {
		t.Fatalf("no SOA rdata in zone file:\n%s", zone)
	}

	return strings.Fields(rest)[0]
}

func TestRenderNamedDisabled(t *testing.T) {
	out := renderByDest(t, strings.Replace(dnsZoneConfig, "enabled: true", "enabled: false", 1))
	assertNoNamed(t, out)
}

func TestRenderNamedAbsent(t *testing.T) {
	out := renderByDest(t, testkit.MinimalConfig)
	assertNoNamed(t, out)
}

func assertNoNamed(t *testing.T, out map[string]string) {
	t.Helper()

	for dest := range out {
		if strings.HasPrefix(dest, "/etc/bind/") {
			t.Errorf("rendered %s while the dns server is not enabled", dest)
		}
	}
}

const dnsForwarderOnly = `version: v3.0.0
hostname: gw
interfaces:
  lan: {select: eth0, addresses: ["10.0.0.1/24"]}
dns:
  server:
    enabled: true
    listen: [iface(lan)]
    allow_from: ["10.0.0.0/24"]
    upstreams: [1.1.1.1]
    forward:
      - {domain: corp.example, servers: [10.0.0.53]}
`

const dnsAuthoritativeOnly = `version: v3.0.0
hostname: gw
interfaces:
  lan: {select: eth0, addresses: ["10.0.0.1/24"]}
dns:
  server:
    enabled: true
    listen: [iface(lan)]
    allow_from: ["10.0.0.0/24"]
    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: {primary: ns1.home.arpa., email: hostmaster@home.arpa}
        records:
          - {name: "@", type: A, value: 10.0.0.1}
          - {name: ns1, type: A, value: 10.0.0.1}
`

const dnsBothRoles = `version: v3.0.0
hostname: gw
interfaces:
  lan: {select: eth0, addresses: ["10.0.0.1/24"]}
dns:
  server:
    enabled: true
    listen: [iface(lan)]
    allow_from: ["10.0.0.0/24"]
    upstreams: [1.1.1.1]
    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: {primary: ns1.home.arpa., email: hostmaster@home.arpa}
        records:
          - {name: "@", type: A, value: 10.0.0.1}
          - {name: ns1, type: A, value: 10.0.0.1}
`

func TestRenderNamedForwarderOnly(t *testing.T) {
	out := namedConf(t, dnsForwarderOnly)

	wantLines(t, "named.conf", out,
		"recursion yes;",
		"allow-recursion { routier_clients; };",
		"forward only;",
	)

	if strings.Contains(out, "type primary;") {
		t.Errorf("forwarder-only must serve no authoritative zones:\n%s", out)
	}
}

func TestRenderNamedAuthoritativeOnlyRefusesRecursion(t *testing.T) {
	out := namedConf(t, dnsAuthoritativeOnly)

	if !strings.Contains(out, "recursion no;") {
		t.Errorf("authoritative-only must refuse recursion, not act as an open resolver:\n%s", out)
	}

	if strings.Contains(out, "recursion yes;") {
		t.Errorf("authoritative-only must not grant recursion:\n%s", out)
	}

	if strings.Contains(out, "forwarders {") || strings.Contains(out, "forward only;") {
		t.Errorf("authoritative-only must emit no forwarding:\n%s", out)
	}

	if !strings.Contains(out, "type primary;") {
		t.Errorf("expected the authoritative zone:\n%s", out)
	}
}

func TestRenderNamedBothRoles(t *testing.T) {
	out := namedConf(t, dnsBothRoles)

	if !strings.Contains(out, "recursion yes;") {
		t.Errorf("both roles must still recurse:\n%s", out)
	}

	if !strings.Contains(out, "forward only;") || !strings.Contains(out, "type primary;") {
		t.Errorf("both roles must forward and serve zones:\n%s", out)
	}
}

func TestDNSModeInference(t *testing.T) {
	cases := map[string]string{
		dnsForwarderOnly:     config.DNSModeForwarder,
		dnsAuthoritativeOnly: config.DNSModeAuthoritative,
		dnsBothRoles:         config.DNSModeBoth,
	}

	for yaml, want := range cases {
		cfg := testkit.LoadCfg(t, yaml)
		if got := cfg.DNS.Server.ResolvedMode(); got != want {
			t.Errorf("inferred mode %q, want %q", got, want)
		}
	}
}

func TestDNSExplicitModeOverridesInference(t *testing.T) {
	cfg := testkit.LoadCfg(t, dnsBothRoles)
	cfg.DNS.Server.Mode = config.DNSModeAuthoritative

	if cfg.DNS.Server.Recurses() {
		t.Fatal("an explicit authoritative mode must disable recursion")
	}
}

const dnsViewsConfig = `version: v3.0.0
hostname: gw
interfaces:
  lan: {select: eth0, addresses: ["10.0.0.1/24"]}
  dmz: {select: eth1, addresses: ["192.0.2.1/24"]}
dns:
  server:
    enabled: true
    listen: [iface(lan), iface(dmz)]
    allow_from: ["10.0.0.0/24", "192.0.2.0/24"]
    upstreams: [1.1.1.1]
    views:
      - name: internal
        match_from: ["10.0.0.0/24"]
        recursion: true
        upstreams: [1.1.1.1]
        zones:
          - name: corp.example
            nameservers: [ns1.corp.example.]
            soa: {primary: ns1.corp.example., email: hostmaster@corp.example}
            records:
              - {name: "@", type: A, value: 10.0.0.1}
              - {name: ns1, type: A, value: 10.0.0.1}
              - {name: wiki, type: A, value: 10.0.0.20}
      - name: external
        match_from: [any]
        recursion: false
        zones:
          - name: corp.example.net
            nameservers: [ns1.corp.example.net.]
            soa: {primary: ns1.corp.example.net., email: hostmaster@corp.example.net}
            records:
              - {name: "@", type: A, value: 192.0.2.1}
              - {name: ns1, type: A, value: 192.0.2.1}
`

func TestRenderNamedViews(t *testing.T) {
	out := renderByDest(t, dnsViewsConfig)
	conf := out[render.NamedConfDest]

	wantLines(t, "named.conf", conf,
		`view "internal" {`,
		`view "external" {`,
		"match-clients {\n        10.0.0.0/24;",
		"match-clients {\n        any;",
	)

	internal := conf[strings.Index(conf, `view "internal"`):strings.Index(conf, `view "external"`)]
	if !strings.Contains(internal, "recursion yes;") {
		t.Errorf("the internal view must recurse:\n%s", internal)
	}

	external := conf[strings.Index(conf, `view "external"`):]
	if !strings.Contains(external, "recursion no;") {
		t.Errorf("the external view must not recurse:\n%s", external)
	}

	if strings.Count(conf, "zone \"corp.example.\" IN {") != 1 {
		t.Errorf("expected the internal zone once:\n%s", conf)
	}

	for _, dest := range []string{
		render.NamedZoneDir + "/corp.example.zone",
		render.NamedZoneDir + "/corp.example.net.zone",
	} {
		if _, ok := out[dest]; !ok {
			t.Errorf("view zone file %s not rendered, got %v", dest, destKeys(out))
		}
	}
}
