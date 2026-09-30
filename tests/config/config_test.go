package configtest

import (
	"github.com/ChevalRouting/routier/tests/testkit"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestConfigLoadValidateSave(t *testing.T) {
	dir := t.TempDir()
	path := testkit.WriteConfig(t, dir, testkit.MinimalConfig)

	cfg, err := config.LoadAndValidate(path, false)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if cfg.Hostname != "rtr1" {
		t.Fatalf("hostname = %q", cfg.Hostname)
	}

	if cfg.Version != config.CurrentVersion {
		t.Fatalf("version = %q, want %q", cfg.Version, config.CurrentVersion)
	}

	out := testkit.WriteConfig(t, dir, "")
	if err := config.Save(out, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded, err := config.Load(out)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	if reloaded.Hostname != cfg.Hostname || len(reloaded.Interfaces) != 1 {
		t.Fatalf("round-trip mismatch: %+v", reloaded)
	}
}

func TestConfigRejectsVersionlessAndBadVersion(t *testing.T) {
	dir := t.TempDir()

	if _, err := config.Load(testkit.WriteConfig(t, dir, "hostname: x\n")); err == nil {
		t.Fatal("expected error for missing version")
	}

	if _, err := config.Load(testkit.WriteConfig(t, dir, "version: v0.0.0\nhostname: x\n")); err == nil {
		t.Fatal("expected error for incompatible major version")
	}
}

func TestMigrationV1ToCurrent(t *testing.T) {
	in := `version: v1.0.0
hostname: rtr
mirrors:
  - name: peer
    url: https://10.0.0.2
    token: secret
ha_keys:
  - name: k
    token: t
nftables:
  chains:
    input:
      rules:
        - ct state established,related accept
        - tcp dport 22 accept
`

	out, changed, err := config.MigrateBytes([]byte(in))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if !changed {
		t.Fatal("expected migration to change the config")
	}

	s := string(out)
	if strings.Contains(s, "mirrors") || strings.Contains(s, "ha_keys") {
		t.Fatalf("legacy HA not dropped:\n%s", s)
	}

	if !strings.Contains(s, "version: v3.0.0") {
		t.Fatalf("not migrated to current version:\n%s", s)
	}

	dir := t.TempDir()
	cfg, err := config.Load(testkit.WriteConfig(t, dir, s))
	if err != nil {
		t.Fatalf("load migrated: %v", err)
	}

	rules := cfg.Nftables.Chains["input"].Rules
	if !strings.Contains(rules, "\n") || !strings.Contains(rules, "dport 22") {
		t.Fatalf("rules not converted to block:\n%q", rules)
	}
}
