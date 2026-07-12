package lifecycletest

import (
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/routier/pkg/managers"
)

func TestPendingApplyConfirmFlow(t *testing.T) {
	restore := managers.SetPendingFile(filepath.Join(t.TempDir(), "pending"))
	defer restore()

	if p, err := managers.PendingStatus(); err != nil || p != nil {
		t.Fatalf("expected no pending, got %+v err %v", p, err)
	}

	if err := managers.SetPending("snap-1", 600, "web"); err != nil {
		t.Fatalf("set pending: %v", err)
	}

	p, err := managers.PendingStatus()
	if err != nil || p == nil {
		t.Fatalf("expected pending, got %+v err %v", p, err)
	}

	if p.SnapID != "snap-1" || p.Timeout != 600 {
		t.Fatalf("unexpected pending: %+v", p)
	}

	if p.Remaining <= 0 || p.Remaining > 600 {
		t.Fatalf("remaining out of range: %d", p.Remaining)
	}

	id, err := managers.ClearPending()
	if err != nil || id != "snap-1" {
		t.Fatalf("clear pending = %q, err %v", id, err)
	}

	if p, _ := managers.PendingStatus(); p != nil {
		t.Fatalf("pending should be gone after confirm: %+v", p)
	}

	if _, err := managers.ClearPending(); err == nil {
		t.Fatal("expected error clearing when nothing pending")
	}
}

func TestWatchdogWaitsWhenNotExpired(t *testing.T) {
	restore := managers.SetPendingFile(filepath.Join(t.TempDir(), "pending"))
	defer restore()

	if err := managers.SetPending("snap-2", 3600, "web"); err != nil {
		t.Fatalf("set pending: %v", err)
	}

	if err := managers.RunWatchdog(); err != nil {
		t.Fatalf("watchdog: %v", err)
	}

	if p, _ := managers.PendingStatus(); p == nil || p.SnapID != "snap-2" {
		t.Fatalf("watchdog should not have cleared a fresh pending apply: %+v", p)
	}
}
