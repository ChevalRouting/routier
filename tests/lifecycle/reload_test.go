package lifecycletest

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/svc"
)

type frrRunner struct {
	mu      sync.Mutex
	running bool
	cmds    [][]string
}

func (r *frrRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmds = append(r.cmds, append([]string{name}, args...))

	joined := strings.Join(args, " ")
	if strings.Contains(joined, "status") {
		if r.running {
			return []byte("status of zebra: running\nstatus of bgpd: running\n"), nil
		}

		return []byte("status of zebra: stopped\n"), nil
	}

	return nil, nil
}

func (r *frrRunner) sawVerb(verb string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.cmds {

		if len(c) >= 3 && c[0] == "rc-service" && c[len(c)-1] == verb && contains(c, "frr") {
			return true
		}
	}

	return false
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}

	return false
}

func TestFRRDaemonsChangeRestartsWhenRunning(t *testing.T) {
	rec := &frrRunner{running: true}
	restore := svc.SetCommandRunner(rec.run)
	defer restore()

	if err := svc.ReloadFromOutputs([]string{"frr/daemons"}, &config.Config{}, false, false); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if !rec.sawVerb("restart") {
		t.Fatalf("expected frr restart on daemons change, commands: %v", rec.cmds)
	}

	if rec.sawVerb("reload") {
		t.Fatalf("must not reload when daemons changed: %v", rec.cmds)
	}
}

func TestFRRDaemonsChangeStartsWhenStopped(t *testing.T) {
	rec := &frrRunner{running: false}
	restore := svc.SetCommandRunner(rec.run)
	defer restore()

	if err := svc.ReloadFromOutputs([]string{"frr/daemons"}, &config.Config{}, false, false); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if !rec.sawVerb("start") {
		t.Fatalf("expected frr start when stopped, commands: %v", rec.cmds)
	}
}

func TestFRRReloadDryRunIsInert(t *testing.T) {
	rec := &frrRunner{running: true}
	restore := svc.SetCommandRunner(rec.run)
	defer restore()

	if err := svc.ReloadFromOutputs([]string{"frr/daemons", "frr/conf"}, &config.Config{}, true, false); err != nil {
		t.Fatalf("dry-run reload: %v", err)
	}

	for _, verb := range []string{"restart", "reload", "start", "stop"} {
		if rec.sawVerb(verb) {
			t.Fatalf("dry-run must not %s frr, got %v", verb, rec.cmds)
		}
	}
}

func (r *frrRunner) firstCmdIndex(name, arg string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, c := range r.cmds {
		if c[0] == name && (arg == "" || contains(c, arg)) {
			return i
		}
	}

	return -1
}

func (r *frrRunner) firstFRRVerbIndex(verb string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, c := range r.cmds {
		if len(c) >= 3 && c[0] == "rc-service" && contains(c, "frr") && c[len(c)-1] == verb {
			return i
		}
	}

	return -1
}

func TestFirewallAndSysctlLoadBeforeFRRRestart(t *testing.T) {
	rec := &frrRunner{running: true}
	restore := svc.SetCommandRunner(rec.run)
	defer restore()

	names := []string{"sysctl/routier.conf", "nftables/routier.nft", "frr/daemons"}
	if err := svc.ReloadFromOutputs(names, &config.Config{}, false, false); err != nil {
		t.Fatalf("reload: %v", err)
	}

	nft := rec.firstCmdIndex("nft", "-f")
	sysctl := rec.firstCmdIndex("sysctl", "")
	frr := rec.firstFRRVerbIndex("restart")

	if nft < 0 || sysctl < 0 || frr < 0 {
		t.Fatalf("missing expected commands (nft=%d sysctl=%d frr=%d): %v", nft, sysctl, frr, rec.cmds)
	}

	if nft > frr {
		t.Fatalf("nftables must load before frr restart, got nft=%d frr=%d: %v", nft, frr, rec.cmds)
	}

	if sysctl > frr {
		t.Fatalf("sysctl must apply before frr restart, got sysctl=%d frr=%d: %v", sysctl, frr, rec.cmds)
	}
}
