package configtest

import (
	"testing"

	"github.com/ChevalRouting/routier/tests/testkit"

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
