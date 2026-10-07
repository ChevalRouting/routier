package persistencetest

import (
	"github.com/ChevalRouting/routier/tests/testkit"
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/state/applylog"
	"github.com/ChevalRouting/routier/pkg/state/backup"
)

func TestApplyLogRecordLifecycle(t *testing.T) {
	restore := applylog.SetDir(t.TempDir())
	defer restore()

	rec := applylog.Start("test", "/etc/routier/config.yml")
	rec.Finish("snap-123", "applied")

	records := applylog.List()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	r := records[0]
	if r.Source != "test" || r.SnapID != "snap-123" || r.Result != "applied" {
		t.Fatalf("unexpected record: %+v", r)
	}

	if _, err := applylog.Read(r.ID); err != nil {
		t.Fatalf("read log: %v", err)
	}

	applylog.MarkConfirmed("snap-123")
	if got := applylog.List(); got[0].ConfirmedAt == nil {
		t.Fatalf("record not marked confirmed: %+v", got[0])
	}
}

func TestBackupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := testkit.WriteConfig(t, dir, testkit.MinimalConfig)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	archive := filepath.Join(dir, "backup.zst")
	if err := backup.Create(archive, cfgPath, cfg); err != nil {
		t.Fatalf("create backup: %v", err)
	}

	restoredPath, err := backup.Restore(archive)
	if err != nil {
		t.Fatalf("restore backup: %v", err)
	}

	restored, err := config.Load(restoredPath)
	if err != nil {
		t.Fatalf("load restored: %v", err)
	}

	if restored.Hostname != "rtr1" {
		t.Fatalf("restored hostname = %q, want rtr1", restored.Hostname)
	}
}
