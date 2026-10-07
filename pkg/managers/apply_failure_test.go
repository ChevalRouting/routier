package managers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/state/applylog"
	"github.com/ChevalRouting/routier/pkg/state/failures"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestPreserveApplyFailureBeforeSnapshot(t *testing.T) {
	defer applylog.SetDir(t.TempDir())()
	defer failures.SetDir(t.TempDir())()
	rec := applylog.Start("web", "config.yml")
	cause := failures.NewValidationError([]types.ArtifactError{{Dest: "/etc/nftables/routier.nft", Line: 12, Message: "syntax error"}})
	err := preserveApplyFailure(rec, "web", "", &config.Config{}, []render.Output{{Dest: "/etc/nftables/routier.nft", Content: "bad rule"}}, cause)
	rec.Finish("", "failed")
	var diagnostic *failures.ApplyError
	if !errors.As(err, &diagnostic) || diagnostic.LogID != rec.ID() || diagnostic.BundleID != rec.ID() {
		t.Fatalf("missing exact diagnostics: %v", err)
	}

	if !errors.Is(err, cause) || cause.BundleID != rec.ID() {
		t.Fatal("original validation error was lost")
	}

	artifact, readErr := failures.ReadArtifact(rec.ID(), "/etc/nftables/routier.nft")
	if readErr != nil || string(artifact) != "bad rule" {
		t.Fatalf("missing artifact: %s, %v", artifact, readErr)
	}

	log, readErr := applylog.Read(rec.ID())
	if readErr != nil || !strings.Contains(log, "syntax error") {
		t.Fatalf("failure missing from log: %s, %v", log, readErr)
	}

	if records := applylog.List(); len(records) != 1 || !records[0].HasBundle {
		t.Fatalf("incorrect history: %+v", records)
	}
}

func TestPreserveFailureKeepsLogWhenBundleCannotBeSaved(t *testing.T) {
	defer applylog.SetDir(t.TempDir())()
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, nil, 0600); err != nil {
		t.Fatal(err)
	}

	defer failures.SetDir(blocked)()
	rec := applylog.Start("cli", "config.yml")
	cause := errors.New("reload kea-dhcp4: exited 1")
	err := preserveApplyFailure(rec, "cli", "", &config.Config{}, nil, cause)
	rec.Finish("", "failed")
	var diagnostic *failures.ApplyError
	if !errors.As(err, &diagnostic) || diagnostic.LogID != rec.ID() || diagnostic.BundleID != "" {
		t.Fatalf("incorrect diagnostics: %v", err)
	}

	if !errors.Is(err, cause) {
		t.Fatal("original runtime error was lost")
	}

	if records := applylog.List(); len(records) != 1 || records[0].HasBundle {
		t.Fatalf("history claims unavailable artifacts: %+v", records)
	}
}
