package svc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/routier/pkg/render"
)

func TestValidateArtifactsParsesToolOutput(t *testing.T) {
	binDir := t.TempDir()
	stub := filepath.Join(binDir, "nft")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	restore := SetCommandRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("/tmp/x/routier.nft:7:3-9: Error: unknown chain\n"), errors.New("exit status 1")
	})

	defer restore()

	outputs := []render.Output{{Name: "nftables/routier.nft", Dest: "/etc/nftables.d/routier.nft", Content: "table inet routier {}\n"}}
	errs := ValidateArtifacts(outputs)
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %d: %+v", len(errs), errs)
	}

	e := errs[0]
	if e.Line != 7 || e.Column != 3 || e.Message != "unknown chain" {
		t.Fatalf("unexpected parsed error: %+v", e)
	}

	if e.Dest != "/etc/nftables.d/routier.nft" || e.Artifact != "nftables/routier.nft" {
		t.Fatalf("dest/artifact not stamped: %+v", e)
	}
}

func TestValidateArtifactsSkipsMissingTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	outputs := []render.Output{{Name: "frr/frr.conf", Dest: "/etc/frr/frr.conf", Content: "hostname r\n"}}
	if errs := ValidateArtifacts(outputs); errs != nil {
		t.Fatalf("want nil when validator binary absent, got %+v", errs)
	}
}

func TestValidateArtifactsBeforeApplyDefersKea(t *testing.T) {
	binDir := t.TempDir()
	for _, bin := range []string{"kea-dhcp4", "kea-dhcp6", "nft"} {
		if err := os.WriteFile(filepath.Join(binDir, bin), []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", binDir)
	var calls []string
	restore := SetCommandRunner(func(_ context.Context, bin string, _ ...string) ([]byte, error) {
		calls = append(calls, bin)
		return nil, nil
	})
	defer restore()
	outputs := []render.Output{
		{Name: "kea/kea-dhcp4.conf", Dest: "/etc/kea/kea-dhcp4.conf", Content: "{}"},
		{Name: "kea/kea-dhcp6.conf", Dest: "/etc/kea/kea-dhcp6.conf", Content: "{}"},
		{Name: "nftables/routier.nft", Dest: "/etc/nftables.d/routier.nft", Content: "table inet routier {}"},
	}
	if errs := ValidateArtifactsBeforeApply(outputs); len(errs) != 0 {
		t.Fatal(errs)
	}
	if len(calls) != 1 || calls[0] != "nft" {
		t.Fatalf("pre-apply validators: %v", calls)
	}
	calls = nil
	if errs := ValidateArtifacts(outputs); len(errs) != 0 {
		t.Fatal(errs)
	}
	if len(calls) != 3 || calls[0] != "kea-dhcp4" || calls[1] != "kea-dhcp6" {
		t.Fatalf("standalone validators: %v", calls)
	}
}
