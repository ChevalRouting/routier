package managers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func usePendingFile(t *testing.T) {
	t.Helper()

	t.Cleanup(SetPendingFile(filepath.Join(t.TempDir(), "pending")))
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestPendingConfigIsWrittenOnlyOnConfirm(t *testing.T) {
	usePendingFile(t)
	configPath := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(configPath, []byte("hostname: confirmed\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := setPending("snap-1", 60, "web", configPath, []byte("hostname: pending\n")); err != nil {
		t.Fatal(err)
	}

	if got := readTestFile(t, configPath); got != "hostname: confirmed\n" {
		t.Fatalf("config changed before confirm: %q", got)
	}

	state, err := takePending()
	if err != nil {
		t.Fatal(err)
	}

	if err := persistPending(state); err != nil {
		t.Fatal(err)
	}

	if got := readTestFile(t, configPath); got != "hostname: pending\n" {
		t.Fatalf("config after confirm = %q", got)
	}

	info, err := os.Stat(configPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("config mode = %v, %v", info.Mode().Perm(), err)
	}
}

func TestDiscardedPendingLeavesConfig(t *testing.T) {
	usePendingFile(t)
	configPath := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(configPath, []byte("hostname: confirmed\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := setPending("snap-1", 60, "web", configPath, []byte("hostname: pending\n")); err != nil {
		t.Fatal(err)
	}

	if _, err := ClearPending(); err != nil {
		t.Fatal(err)
	}

	if got := readTestFile(t, configPath); got != "hostname: confirmed\n" {
		t.Fatalf("discarding a pending apply changed the config: %q", got)
	}
}

func TestPendingWithoutConfigPersistsNothing(t *testing.T) {
	usePendingFile(t)

	if err := SetPending("snap-1", 60, "cli"); err != nil {
		t.Fatal(err)
	}

	state, err := takePending()
	if err != nil {
		t.Fatal(err)
	}

	if state.Config != nil || state.ConfigPath != "" {
		t.Fatalf("operator apply should not carry a config: %+v", state)
	}

	if err := persistPending(state); err != nil {
		t.Fatal(err)
	}
}

func TestRequireNoPending(t *testing.T) {
	usePendingFile(t)

	if err := RequireNoPending(); err != nil {
		t.Fatalf("no pending apply: %v", err)
	}

	if err := SetPending("snap-1", 60, "web"); err != nil {
		t.Fatal(err)
	}

	if err := RequireNoPending(); !errors.Is(err, ErrApplyPending) {
		t.Fatalf("err = %v, want ErrApplyPending", err)
	}
}

func TestWithDDNSKeyCarriesGeneratedKey(t *testing.T) {
	persist := []byte("version: v3.0.0\ndhcp:\n  ddns:\n    enabled: true\n")
	cfg := &config.Config{DHCP: &config.DHCP{DDNS: &config.DHCPDDNS{Enabled: true}}}
	if !config.EnsureDDNSKey(cfg) {
		t.Fatal("expected a generated key")
	}

	out, err := withDDNSKey(persist, cfg)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := config.LoadBytes(out)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.DHCP.DDNS.Key != cfg.DHCP.DDNS.Key || loaded.DHCP.DDNS.Algorithm != cfg.DHCP.DDNS.Algorithm {
		t.Fatalf("persisted ddns = %+v, want key and algorithm from the applied config", loaded.DHCP.DDNS)
	}
}
