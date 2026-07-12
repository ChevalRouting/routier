package lifecycletest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ChevalRouting/routier/pkg/svc"
)

type recordingRunner struct {
	mu   sync.Mutex
	cmds [][]string
}

func (r *recordingRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmds = append(r.cmds, append([]string{name}, args...))
	return []byte("status: started\n"), nil
}

func (r *recordingRunner) ran(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.cmds {
		if len(c) > 0 && c[0] == name {
			return true
		}
	}

	return false
}

func TestServiceReloadIsMockable(t *testing.T) {
	rec := &recordingRunner{}
	restore := svc.SetCommandRunner(rec.run)
	defer restore()

	actions := []svc.Action{
		{Desc: "reload sshd", Args: []string{"rc-service", "sshd", "reload"}, Timeout: time.Second},
		{Desc: "restart conntrackd", Args: []string{"rc-service", "conntrackd", "restart"}},
	}

	if err := svc.Reload(actions, false); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if !rec.ran("rc-service") {
		t.Fatalf("expected rc-service to be invoked through the mock, got %v", rec.cmds)
	}

	if len(rec.cmds) != 2 {
		t.Fatalf("expected 2 mocked commands, got %v", rec.cmds)
	}
}

func TestServiceRunningUsesRunner(t *testing.T) {
	restore := svc.SetCommandRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("status: started\n"), nil
	})
	defer restore()

	if !svc.ServiceRunning("anything") {
		t.Fatal("expected mocked service to report running")
	}

	restore2 := svc.SetCommandRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("status: stopped\n"), nil
	})
	defer restore2()

	if svc.ServiceRunning("anything") {
		t.Fatal("expected mocked service to report stopped")
	}
}
