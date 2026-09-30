package clitest

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/tests/testkit"
)

func TestCLIVersion(t *testing.T) {
	bin := testkit.BuildCLI(t)

	out, err := testkit.RunCLI(t, bin, "version")
	if err != nil {
		t.Fatalf("version: %v\n%s", err, out)
	}

	if strings.TrimSpace(out) == "" {
		t.Fatal("version printed nothing")
	}
}

func TestCLIValidate(t *testing.T) {
	bin := testkit.BuildCLI(t)
	cfg := testkit.WriteConfig(t, t.TempDir(), testkit.MinimalConfig)

	if out, err := testkit.RunCLI(t, bin, "validate", "--skip-resolve", cfg); err != nil {
		t.Fatalf("validate valid config: %v\n%s", err, out)
	}

	bad := testkit.WriteConfig(t, t.TempDir(), "version: v3.0.0\nhostname: x\nwireguard:\n  wg0:\n    addresses: [\"10.0.0.1/24\"]\n")
	if _, err := testkit.RunCLI(t, bin, "validate", "--skip-resolve", bad); err == nil {
		t.Fatal("expected validate to fail on invalid config")
	}
}

func TestCLIMigrate(t *testing.T) {
	bin := testkit.BuildCLI(t)
	cfg := testkit.WriteConfig(t, t.TempDir(), "version: v1.0.0\nhostname: x\nmirrors:\n  - name: m\n    url: https://h\n    token: t\n")

	out, err := testkit.RunCLI(t, bin, "migrate", "--dry-run", cfg)
	if err != nil {
		t.Fatalf("migrate dry-run: %v\n%s", err, out)
	}
}

func TestCLIDNSCommandsExist(t *testing.T) {
	bin := testkit.BuildCLI(t)

	out, err := testkit.RunCLI(t, bin, "dns", "--help")
	if err != nil {
		t.Fatalf("dns --help: %v\n%s", err, out)
	}

	for _, sub := range []string{"stats", "zones", "query", "flush", "reload"} {
		if !strings.Contains(out, sub) {
			t.Errorf("routier dns is missing the %q subcommand:\n%s", sub, out)
		}
	}
}
