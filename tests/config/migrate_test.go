package configtest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestMigrateNftablesRulesV1ToBlock(t *testing.T) {
	in := `version: v1.0.0
hostname: rtr
nftables:
  chains:
    input:
      policy: drop
      rules:
        - iifname "lo" accept
        - tcp dport 22 accept
    forward:
      rules:
        - ct state established,related accept
`
	out, changed, err := config.MigrateBytes([]byte(in))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if !changed {
		t.Fatal("expected changed")
	}

	var raw map[string]any
	if err := yaml.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if raw["version"] != config.CurrentVersion {
		t.Fatalf("version not bumped: %v", raw["version"])
	}

	cfg := &config.Config{}
	if err := yaml.Unmarshal(out, cfg); err != nil {
		t.Fatalf("typed unmarshal: %v", err)
	}

	if cfg.Nftables.Chains["input"].Rules != "iifname \"lo\" accept\ntcp dport 22 accept" {
		t.Fatalf("unexpected joined rules: %q", cfg.Nftables.Chains["input"].Rules)
	}

	if !strings.Contains(cfg.Nftables.Chains["forward"].Rules, "established,related") {
		t.Fatalf("forward rules missing: %q", cfg.Nftables.Chains["forward"].Rules)
	}
}

func TestMigrateKeepsDNSServer(t *testing.T) {
	in := `version: v1.0.0
hostname: rtr
dns:
  nameservers:
    - 127.0.0.1
  search:
    - 42.school
  server:
    enabled: true
    listen:
      - 127.0.0.1
      - iface(region)
      - vips(region)
    allow_from:
      - 10.170.32.0/24
    allow_inbound:
      - region
    upstreams:
      - 1.1.1.1
    forward:
      - domain: 42.school
        servers:
          - 10.255.0.54
    cache:
      disabled: true
    zones:
      - name: home.arpa
        nameservers:
          - ns1.home.arpa.
        soa:
          email: hostmaster@home.arpa
        records:
          - name: "@"
            type: A
            value: 10.0.0.1
`

	out, changed, err := config.MigrateBytes([]byte(in))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if !changed {
		t.Fatal("expected the version bump to change the config")
	}

	if !strings.Contains(string(out), "version: "+config.CurrentVersion) {
		t.Fatalf("not migrated to current version:\n%s", out)
	}

	before, after := dnsSubtree(t, []byte(in)), dnsSubtree(t, out)
	if before != after {
		t.Fatalf("dns subtree rewritten by migration:\nbefore:\n%s\nafter:\n%s", before, after)
	}

	cfg := &config.Config{}
	if err := yaml.Unmarshal(out, cfg); err != nil {
		t.Fatalf("typed unmarshal: %v", err)
	}

	if cfg.DNS == nil || cfg.DNS.Server == nil || !cfg.DNS.Server.Enabled {
		t.Fatalf("dns.server lost: %+v", cfg.DNS)
	}

	if len(cfg.DNS.Server.Listen) != 3 || len(cfg.DNS.Server.Zones) != 1 || len(cfg.DNS.Server.Zones[0].Records) != 1 {
		t.Fatalf("dns.server truncated: %+v", cfg.DNS.Server)
	}
}

func dnsSubtree(t *testing.T, doc []byte) string {
	t.Helper()

	var raw map[string]any
	if err := yaml.Unmarshal(doc, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := yaml.Marshal(raw["dns"])
	if err != nil {
		t.Fatalf("marshal dns: %v", err)
	}

	return string(out)
}

func TestMigrateIdempotentAtCurrent(t *testing.T) {
	in := "version: " + config.CurrentVersion + "\nhostname: rtr\n"
	_, changed, err := config.MigrateBytes([]byte(in))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if changed {
		t.Fatal("current-version config should not change")
	}
}
