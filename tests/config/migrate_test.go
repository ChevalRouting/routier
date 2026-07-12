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
