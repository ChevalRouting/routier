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
