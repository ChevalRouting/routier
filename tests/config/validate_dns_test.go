package configtest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/tests/testkit"
)

const dnsBase = `version: v3.0.0
hostname: gw
interfaces:
  region:
    select: name=eth1
    addresses: ["10.170.32.2/24"]
  workstations:
    select: name=eth2
    addresses: ["10.170.48.2/20"]
  external:
    select: name=eth3
    addresses: ["10.170.33.2/24"]
  wan:
    select: name=eth0
    addresses: [dhcp]
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
  server:
    enabled: true
`

const dnsServerBase = dnsBase + `    listen: ["127.0.0.1"]
    allow_from: ["10.0.0.0/8"]
`

type dnsCase struct {
	yaml    string
	wantErr string
}

func TestValidateDNSServerValid(t *testing.T) {
	cases := map[string]string{
		"forwarding target": `version: v3.0.0
hostname: gw
interfaces:
  region:
    select: name=eth1
    addresses: ["10.170.32.2/24"]
  workstations:
    select: name=eth2
    addresses: ["10.170.48.2/20"]
  external:
    select: name=eth3
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
`,
		"authoritative zones": `version: v3.0.0
hostname: gw
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.1/24"]
  guest:
    type: vlan
    select: lan
    addresses: ["10.0.100.1/24"]
    vlan:
      id: 100
dns:
  server:
    enabled: true
    listen: [iface(lan), iface(guest)]
    allow_from: [10.0.0.0/24, 10.0.100.0/24]
    allow_inbound: [lan, guest]
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
          - { name: _sip._tcp, type: SRV, value: "10 5060 sip.home.arpa.", priority: 10 }
      - name: 0.0.10.in-addr.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "1",  type: PTR, value: gw.home.arpa. }
          - { name: "10", type: PTR, value: nas.home.arpa. }
      - name: secondary.example
        primaries: [10.255.0.54]
`,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if msg := validateErr(t, yaml); msg != "" {
				t.Fatalf("expected valid dns server config, got: %s", msg)
			}
		})
	}
}

func TestValidateDNSServer(t *testing.T) {
	cases := map[string]dnsCase{
		"unknown listen interface": {
			yaml: dnsBase + `    listen: ["iface(nope)"]
    allow_from: ["10.0.0.0/8"]
`,
			wantErr: `iface(nope)`,
		},
		"listen on an interface without vips": {
			yaml: dnsBase + `    listen: ["vips(external)"]
    allow_from: ["10.0.0.0/8"]
`,
			wantErr: "no vrrp vips",
		},
		"listen on a dhcp only interface": {
			yaml: dnsBase + `    listen: ["iface(wan)"]
    allow_from: ["10.0.0.0/8"]
`,
			wantErr: "no static address",
		},
		"malformed listen address": {
			yaml: dnsBase + `    listen: ["10.170.32.999"]
    allow_from: ["10.0.0.0/8"]
`,
			wantErr: "is not an IP address",
		},
		"missing allow_from": {
			yaml: dnsBase + `    listen: ["127.0.0.1"]
`,
			wantErr: "allow_from is required",
		},
		"bad allow_from cidr": {
			yaml: dnsBase + `    listen: ["127.0.0.1"]
    allow_from: ["10.0.0.0/33"]
`,
			wantErr: "is not a valid IP or CIDR",
		},
		"unknown allow_inbound interface": {
			yaml: dnsServerBase + `    allow_inbound: [nope]
`,
			wantErr: "not found in interfaces",
		},
		"bad upstream": {
			yaml: dnsServerBase + `    upstreams: ["not-an-ip"]
`,
			wantErr: "dns.server.upstreams[0]",
		},
		"duplicate forward domain": {
			yaml: dnsServerBase + `    forward:
      - domain: 42.school
        servers: [10.255.0.54]
      - domain: 42.school.
        servers: [10.255.0.55]
`,
			wantErr: "duplicate forward domain",
		},
		"forward without servers": {
			yaml: dnsServerBase + `    forward:
      - domain: 42.school
`,
			wantErr: "at least one server is required",
		},
		"forward domain is also a zone": {
			yaml: dnsServerBase + `    forward:
      - domain: home.arpa
        servers: [10.255.0.54]
    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
          - { name: ns1, type: A, value: 10.0.0.1 }
`,
			wantErr: "is also an authoritative zone",
		},
		"duplicate zone name": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
      - name: HOME.ARPA.
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: "@", type: A, value: 10.0.0.2 }
`,
			wantErr: "duplicate zone name",
		},
		"zone with records and primaries": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        primaries: [10.255.0.54]
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
          - { name: ns1, type: A, value: 10.0.0.1 }
`,
			wantErr: "mutually exclusive",
		},
		"zone without nameservers": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
          - { name: ns1, type: A, value: 10.0.0.1 }
`,
			wantErr: "nameservers is required",
		},
		"zone without soa email": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
          - { name: ns1, type: A, value: 10.0.0.1 }
`,
			wantErr: "soa.email is required",
		},
		"record name outside the zone": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: nas.example.com., type: A, value: 10.0.0.10 }
`,
			wantErr: "an absolute name inside it",
		},
		"a record holding a v6 address": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: nas, type: A, value: "2001:db8::10" }
`,
			wantErr: "is not an IPv4 address",
		},
		"aaaa record holding a v4 address": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: nas, type: AAAA, value: 10.0.0.10 }
`,
			wantErr: "is not an IPv6 address",
		},
		"multiple cname targets for one owner": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: lb, type: CNAME, value: lb-1.home.arpa. }
          - { name: lb, type: CNAME, value: lb-2.home.arpa. }
`,
			wantErr: "only one CNAME target is allowed per owner",
		},
		"cname colliding with an a record": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: nas, type: A,     value: 10.0.0.10 }
          - { name: nas, type: CNAME, value: gw.home.arpa. }
`,
			wantErr: "CNAME alongside other records",
		},
		"priority on an a record": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: nas, type: A, value: 10.0.0.10, priority: 10 }
`,
			wantErr: "priority is only supported on MX and SRV",
		},
		"unknown record type": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { email: hostmaster@home.arpa }
        records:
          - { name: nas, type: SPF, value: "v=spf1 -all" }
`,
			wantErr: "must be one of",
		},
		"min ttl above max ttl": {
			yaml: dnsServerBase + `    cache:
      min_ttl: 3600
      max_ttl: 60
`,
			wantErr: "must not exceed max_ttl",
		},
		"bad cache size": {
			yaml: dnsServerBase + `    cache:
      size: 64mb
`,
			wantErr: "must be a byte count",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) { testValidateDNSServerCallback(&tc, t) })
	}
}

func TestValidateDNSServerDisabledSkipsChecks(t *testing.T) {
	yaml := `version: v3.0.0
hostname: gw
dns:
  server:
    enabled: false
    listen: ["iface(nope)", "garbage"]
    upstreams: ["not-an-ip"]
    allow_inbound: [nope]
    cache:
      size: 64mb
      min_ttl: 3600
      max_ttl: 60
    forward:
      - domain: "not a domain"
    zones:
      - name: home.arpa
        records:
          - { name: nas, type: SPF, value: 10.0.0.10, priority: 3 }
      - name: home.arpa
        records:
          - { name: nas, type: A, value: "2001:db8::10" }
`

	if msg := validateErr(t, yaml); msg != "" {
		t.Fatalf("disabled dns server should skip validation, got: %s", msg)
	}
}

func TestFullConfigDNSServerRoundTrips(t *testing.T) {
	dir := t.TempDir()

	cfg, err := config.LoadAndValidate(testkit.WriteConfig(t, dir, testkit.FullConfig), false)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	out := testkit.WriteConfig(t, dir, "")
	if err := config.Save(out, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded, err := config.Load(out)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	assertDNSServer(t, "yaml", reloaded.DNS.Server)

	data, err := json.Marshal(cfg.DNS.Server)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fromJSON config.DNSServer
	if err := json.Unmarshal(data, &fromJSON); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	assertDNSServer(t, "json", &fromJSON)
}

func assertDNSServer(t *testing.T, kind string, s *config.DNSServer) {
	t.Helper()

	if s == nil {
		t.Fatalf("%s: dns.server lost", kind)
	}

	if !s.Enabled || s.Threads != 2 || !s.LogQueries {
		t.Fatalf("%s: scalars lost: %+v", kind, s)
	}

	if len(s.Listen) != 3 || s.Listen[1] != "iface(lan)" || s.Listen[2] != "vips(lan)" {
		t.Fatalf("%s: listen lost: %v", kind, s.Listen)
	}

	if len(s.AllowInbound) != 1 || len(s.Upstreams) != 2 || len(s.AllowFrom) != 1 {
		t.Fatalf("%s: acl lost: %+v", kind, s)
	}

	if len(s.Forward) != 1 || s.Forward[0].Domain != "corp.example.net" || len(s.Forward[0].Servers) != 1 {
		t.Fatalf("%s: forward lost: %+v", kind, s.Forward)
	}

	if s.Cache == nil || s.Cache.Size != "64m" || s.Cache.MinTTL != 60 || !s.Cache.Prefetch {
		t.Fatalf("%s: cache lost: %+v", kind, s.Cache)
	}

	if len(s.Zones) != 2 || s.Zones[0].Name != "example.net" || s.Zones[0].TTL != 300 {
		t.Fatalf("%s: zones lost: %+v", kind, s.Zones)
	}

	z := s.Zones[0]
	if z.SOA == nil || z.SOA.Email != "hostmaster@example.net" || z.SOA.Expire != 604800 {
		t.Fatalf("%s: soa lost: %+v", kind, z.SOA)
	}

	if len(z.Records) != 6 || z.Records[4].Type != "MX" || z.Records[4].Priority != 10 {
		t.Fatalf("%s: records lost: %+v", kind, z.Records)
	}
}

func TestValidateDNSMode(t *testing.T) {
	zone := `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
          - { name: ns1, type: A, value: 10.0.0.1 }
`

	cases := map[string]dnsCase{
		"unknown mode": {
			yaml:    dnsServerBase + "    mode: recursive\n    upstreams: [1.1.1.1]\n",
			wantErr: `mode "recursive" must be one of`,
		},
		"authoritative with upstreams": {
			yaml:    dnsServerBase + "    mode: authoritative\n    upstreams: [1.1.1.1]\n" + zone,
			wantErr: "remove upstreams and forward",
		},
		"authoritative with forward": {
			yaml: dnsServerBase + `    mode: authoritative
    forward:
      - { domain: corp.example, servers: [10.0.0.53] }
` + zone,
			wantErr: "remove upstreams and forward",
		},
		"authoritative without zones": {
			yaml:    dnsServerBase + "    mode: authoritative\n",
			wantErr: "requires at least one zone",
		},
		"forwarder with zones": {
			yaml:    dnsServerBase + "    mode: forwarder\n    upstreams: [1.1.1.1]\n" + zone,
			wantErr: "does not serve zones",
		},
		"both without zones": {
			yaml:    dnsServerBase + "    mode: both\n    upstreams: [1.1.1.1]\n",
			wantErr: "requires at least one zone",
		},
		"both without upstreams": {
			yaml:    dnsServerBase + "    mode: both\n" + zone,
			wantErr: "requires upstreams or forward",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) { testValidateDNSModeCallback(&c, t) })
	}
}

func TestValidateDNSModeAccepted(t *testing.T) {
	zone := `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
          - { name: ns1, type: A, value: 10.0.0.1 }
`

	cases := map[string]string{
		"forwarder":     dnsServerBase + "    mode: forwarder\n    upstreams: [1.1.1.1]\n",
		"authoritative": dnsServerBase + "    mode: authoritative\n" + zone,
		"both":          dnsServerBase + "    mode: both\n    upstreams: [1.1.1.1]\n" + zone,
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) {
			if msg := validateErr(t, yaml); msg != "" {
				t.Fatalf("expected a valid config, got: %s", msg)
			}
		})
	}
}

func TestValidateDNSZoneNameserverGlue(t *testing.T) {
	missing := dnsServerBase + `    mode: authoritative
    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
`
	msg := validateErr(t, missing)
	if !strings.Contains(msg, "has no A or AAAA record") {
		t.Fatalf("expected missing-glue error, got: %s", msg)
	}

	present := dnsServerBase + `    mode: authoritative
    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "@",   type: A, value: 10.0.0.1 }
          - { name: ns1,   type: A, value: 10.0.0.1 }
`
	if msg := validateErr(t, present); msg != "" {
		t.Fatalf("in-zone nameserver with glue rejected: %s", msg)
	}

	external := dnsServerBase + `    mode: authoritative
    zones:
      - name: home.arpa
        nameservers: [ns1.example.net.]
        soa: { primary: ns1.example.net., email: hostmaster@home.arpa }
        records:
          - { name: "@", type: A, value: 10.0.0.1 }
`
	if msg := validateErr(t, external); msg != "" {
		t.Fatalf("out-of-zone nameserver needs no glue, got: %s", msg)
	}
}

func TestValidateDNSViews(t *testing.T) {
	viewZone := `        zones:
          - name: corp.example
            nameservers: [ns1.corp.example.]
            soa: { primary: ns1.corp.example., email: hostmaster@corp.example }
            records:
              - { name: "@",  type: A, value: 10.0.0.1 }
              - { name: ns1,  type: A, value: 10.0.0.1 }
`

	cases := map[string]dnsCase{
		"views alongside top-level zones": {
			yaml: dnsServerBase + `    zones:
      - name: home.arpa
        nameservers: [ns1.home.arpa.]
        soa: { primary: ns1.home.arpa., email: hostmaster@home.arpa }
        records:
          - { name: "@",   type: A, value: 10.0.0.1 }
          - { name: ns1,   type: A, value: 10.0.0.1 }
    views:
      - name: internal
        match_from: ["10.0.0.0/24"]
` + viewZone,
			wantErr: "mutually exclusive",
		},
		"duplicate view name": {
			yaml: dnsServerBase + `    views:
      - name: internal
        match_from: ["10.0.0.0/24"]
` + viewZone + `      - name: internal
        match_from: [any]
` + viewZone,
			wantErr: "duplicate view name",
		},
		"view without a name": {
			yaml: dnsServerBase + `    views:
      - match_from: [any]
` + viewZone,
			wantErr: "name is required",
		},
		"view with a bad match_from": {
			yaml: dnsServerBase + `    views:
      - name: internal
        match_from: ["not-a-cidr"]
` + viewZone,
			wantErr: "not a valid IP, CIDR or BIND acl keyword",
		},
		"empty view": {
			yaml: dnsServerBase + `    views:
      - name: internal
        match_from: [any]
`,
			wantErr: "needs at least one zone or forward entry",
		},
		"view zone missing glue": {
			yaml: dnsServerBase + `    views:
      - name: internal
        match_from: [any]
        zones:
          - name: corp.example
            nameservers: [ns1.corp.example.]
            soa: { primary: ns1.corp.example., email: hostmaster@corp.example }
            records:
              - { name: "@", type: A, value: 10.0.0.1 }
`,
			wantErr: "has no A or AAAA record",
		},
		"view zone bad record": {
			yaml: dnsServerBase + `    views:
      - name: internal
        match_from: [any]
        zones:
          - name: corp.example
            nameservers: [ns1.corp.example.]
            soa: { primary: ns1.corp.example., email: hostmaster@corp.example }
            records:
              - { name: "@",  type: A, value: "2001:db8::1" }
              - { name: ns1,  type: A, value: 10.0.0.1 }
`,
			wantErr: "views[0].zones[0]",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) { testValidateDNSViewsCallback(&c, t) })
	}
}

func TestValidateDNSViewsAccepted(t *testing.T) {
	yaml := dnsServerBase + `    upstreams: [1.1.1.1]
    views:
      - name: internal
        match_from: ["10.0.0.0/24"]
        recursion: true
        upstreams: [1.1.1.1]
        zones:
          - name: corp.example
            nameservers: [ns1.corp.example.]
            soa: { primary: ns1.corp.example., email: hostmaster@corp.example }
            records:
              - { name: "@",  type: A, value: 10.0.0.1 }
              - { name: ns1,  type: A, value: 10.0.0.1 }
      - name: external
        match_from: [any]
        recursion: false
        zones:
          - name: corp.example.net
            nameservers: [ns1.corp.example.net.]
            soa: { primary: ns1.corp.example.net., email: hostmaster@corp.example.net }
            records:
              - { name: "@",  type: A, value: 192.0.2.1 }
              - { name: ns1,  type: A, value: 192.0.2.1 }
`

	if msg := validateErr(t, yaml); msg != "" {
		t.Fatalf("valid views config rejected: %s", msg)
	}
}

func testValidateDNSServerCallback(tc *dnsCase, t *testing.T) {
	msg := validateErr(t, (*tc).yaml)
	if !strings.Contains(msg, (*tc).wantErr) {
		t.Fatalf("expected an error containing %q, got: %q", (*tc).wantErr, msg)
	}
}

func testValidateDNSModeCallback(c *dnsCase, t *testing.T) {
	msg := validateErr(t, (*c).yaml)
	if msg == "" {
		t.Fatal("expected a validation error")
	}

	if !strings.Contains(msg, (*c).wantErr) {
		t.Fatalf("expected %q in: %s", (*c).wantErr, msg)
	}
}

func testValidateDNSViewsCallback(c *dnsCase, t *testing.T) {
	msg := validateErr(t, (*c).yaml)
	if msg == "" {
		t.Fatal("expected a validation error")
	}

	if !strings.Contains(msg, (*c).wantErr) {
		t.Fatalf("expected %q in: %s", (*c).wantErr, msg)
	}
}
