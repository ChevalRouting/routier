package lifecycletest

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/ChevalRouting/routier/pkg/artifacterr"
	"github.com/ChevalRouting/routier/pkg/daemon/bind"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/svc"
)

type namedRunner struct {
	mu           sync.Mutex
	running      bool
	checkconfErr bool
	cmds         [][]string
	control      []string
	controlErr   bool
}

func (r *namedRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cmds = append(r.cmds, append([]string{name}, args...))

	joined := strings.Join(args, " ")

	if name == "named-checkconf" {
		if r.checkconfErr {
			return []byte("/etc/bind/named.conf:12: error: syntax error"), errors.New("exit status 1")
		}

		return []byte("named-checkconf: no errors in /etc/bind/named.conf"), nil
	}

	if name == "rc-service" && strings.Contains(joined, "status") {
		if strings.Contains(joined, "named") && r.running {
			return []byte("status: started\n"), nil
		}

		return []byte("status: stopped\n"), nil
	}

	return nil, nil
}

func (r *namedRunner) rndc(_ string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var cmd []string
	for i := 0; i < len(args); i++ {
		if args[i] == "-s" || args[i] == "-p" || args[i] == "-k" {
			i++
			continue
		}

		cmd = append(cmd, args[i])
	}

	r.control = append(r.control, strings.Join(cmd, " "))

	if r.controlErr {
		return []byte("rndc: connect failed"), errors.New("exit status 1")
	}

	return []byte("ok"), nil
}

func (r *namedRunner) sawNamedVerb(verb string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.cmds {
		if c[0] == "rc-service" && c[len(c)-1] == verb && contains(c, "named") {
			return true
		}
	}

	return false
}

func (r *namedRunner) sawCmd(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.cmds {
		if c[0] == name {
			return true
		}
	}

	return false
}

func (r *namedRunner) controlCmds() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string{}, r.control...)
}

func (r *namedRunner) sawControl(prefix string) bool {
	for _, c := range r.controlCmds() {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}

	return false
}

func dnsServerConfig() *config.Config {
	return &config.Config{
		Hostname: "gw",
		DNS: &config.DNS{
			Server: &config.DNSServer{
				Enabled:   true,
				Listen:    []string{"127.0.0.1"},
				AllowFrom: []string{"127.0.0.1/32"},
				Upstreams: []string{"1.1.1.1"},
				Zones:     []config.DNSZone{{Name: "42.school"}},
			},
		},
	}
}

func TestNamedConfChangeReconfigures(t *testing.T) {
	rec := &namedRunner{running: true}
	defer svc.SetCommandRunner(rec.run)()
	defer bind.SetRunner(rec.rndc)()

	if err := svc.ReloadFromOutputs([]string{render.NamedConfName}, dnsServerConfig(), false, true); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if !rec.sawCmd("named-checkconf") {
		t.Fatalf("expected named-checkconf before reload, commands: %v", rec.cmds)
	}

	if !rec.sawControl("reconfig") {
		t.Fatalf("expected reconfig, control: %v", rec.controlCmds())
	}

	if rec.sawNamedVerb("restart") {
		t.Fatalf("must not restart when reconfig succeeded: %v", rec.cmds)
	}
}

func TestNamedZoneOnlyChangeReloadsZone(t *testing.T) {
	rec := &namedRunner{running: true}
	defer svc.SetCommandRunner(rec.run)()
	defer bind.SetRunner(rec.rndc)()

	names := []string{render.NamedZoneName + "42.school.zone"}
	if err := svc.ReloadFromOutputs(names, dnsServerConfig(), false, true); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if !rec.sawControl("reload 42.school") {
		t.Fatalf("expected reload 42.school, control: %v", rec.controlCmds())
	}

	for _, unwanted := range []string{"reconfig", "reload"} {
		for _, c := range rec.controlCmds() {
			if c == unwanted {
				t.Fatalf("zone-only change must not %s, control: %v", unwanted, rec.controlCmds())
			}
		}
	}

	if rec.sawNamedVerb("restart") || rec.sawNamedVerb("reload") {
		t.Fatalf("zone-only change must not restart or rc-service reload: %v", rec.cmds)
	}
}

func TestNamedControlFailureFallsBackToRestart(t *testing.T) {
	rec := &namedRunner{running: true, controlErr: true}
	defer svc.SetCommandRunner(rec.run)()
	defer bind.SetRunner(rec.rndc)()

	if err := svc.ReloadFromOutputs([]string{render.NamedConfName}, dnsServerConfig(), false, true); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if !rec.sawNamedVerb("reload") {
		t.Fatalf("expected rc-service reload fallback, commands: %v", rec.cmds)
	}
}

func TestNamedStartsWhenStopped(t *testing.T) {
	rec := &namedRunner{running: false}
	defer svc.SetCommandRunner(rec.run)()
	defer bind.SetRunner(rec.rndc)()

	if err := svc.ReloadFromOutputs([]string{render.NamedConfName}, dnsServerConfig(), false, true); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if !rec.sawNamedVerb("start") {
		t.Fatalf("expected named start when stopped, commands: %v", rec.cmds)
	}

	if len(rec.controlCmds()) != 0 {
		t.Fatalf("must not issue control commands to a stopped daemon: %v", rec.controlCmds())
	}
}

func TestNamedValidationFailureAbortsApply(t *testing.T) {
	rec := &namedRunner{running: true, checkconfErr: true}
	defer svc.SetCommandRunner(rec.run)()
	defer bind.SetRunner(rec.rndc)()

	err := svc.ReloadFromOutputs([]string{render.NamedConfName}, dnsServerConfig(), false, true)
	if err == nil {
		t.Fatal("expected a failed named-checkconf to abort the apply")
	}

	if !strings.Contains(err.Error(), "config invalid") {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rec.controlCmds()) != 0 {
		t.Fatalf("must not reload after a validation failure: %v", rec.controlCmds())
	}

	for _, verb := range []string{"reload", "restart", "start"} {
		if rec.sawNamedVerb(verb) {
			t.Fatalf("must not %s after a validation failure: %v", verb, rec.cmds)
		}
	}
}

func TestNamedDryRunIsInert(t *testing.T) {
	rec := &namedRunner{running: true}
	defer svc.SetCommandRunner(rec.run)()
	defer bind.SetRunner(rec.rndc)()

	names := []string{render.NamedConfName, render.NamedZoneName + "42.school.zone"}
	if err := svc.ReloadFromOutputs(names, dnsServerConfig(), true, true); err != nil {
		t.Fatalf("dry-run reload: %v", err)
	}

	if rec.sawCmd("named-checkconf") {
		t.Fatalf("dry-run must not run named-checkconf: %v", rec.cmds)
	}

	if len(rec.controlCmds()) != 0 {
		t.Fatalf("dry-run must not issue control commands: %v", rec.controlCmds())
	}

	for _, verb := range []string{"reload", "restart", "start"} {
		if rec.sawNamedVerb(verb) {
			t.Fatalf("dry-run must not %s named, got %v", verb, rec.cmds)
		}
	}
}

func TestNamedZoneReloadDoesNotReconfigure(t *testing.T) {
	rec := &namedRunner{running: true}
	defer svc.SetCommandRunner(rec.run)()
	defer bind.SetRunner(rec.rndc)()

	cfg := dnsServerConfig()
	cfg.DNS.Server.Cache = &config.DNSCache{Size: "64m"}

	names := []string{render.NamedZoneName + "42.school.zone"}
	if err := svc.ReloadFromOutputs(names, cfg, false, true); err != nil {
		t.Fatalf("reload: %v", err)
	}

	for _, c := range rec.controlCmds() {
		if c == "reconfig" {
			t.Fatalf("a zone-only change must not reconfigure named: %v", rec.controlCmds())
		}
	}
}

func TestNamedReloadsBeforeKea(t *testing.T) {
	rec := &namedRunner{running: true}
	defer svc.SetCommandRunner(rec.run)()
	defer bind.SetRunner(rec.rndc)()

	names := []string{render.NamedConfName, "kea/kea-dhcp4.conf"}
	_ = svc.ReloadFromOutputs(names, dnsServerConfig(), false, true)

	rec.mu.Lock()
	defer rec.mu.Unlock()

	namedAt, keaAt := -1, -1
	for i, c := range rec.cmds {
		if namedAt == -1 && c[0] == "named-checkconf" {
			namedAt = i
		}

		if keaAt == -1 && c[0] == "kea-dhcp4" {
			keaAt = i
		}
	}

	if namedAt == -1 || keaAt == -1 {
		t.Skipf("both services must be exercised, commands: %v", rec.cmds)
	}

	if namedAt > keaAt {
		t.Fatalf("named must be applied before kea, commands: %v", rec.cmds)
	}
}

func TestValidateArtifactsChecksZoneFiles(t *testing.T) {
	if _, err := exec.LookPath("named-checkzone"); err != nil {
		t.Skip("named-checkzone not installed")
	}

	good := render.Output{
		Name: render.NamedZoneName + "42.school.zone",
		Dest: render.NamedZoneDir + "/42.school.zone",
		Content: `$ORIGIN 42.school.
$TTL 300
@ IN SOA ns.42.school. hostmaster.42.school. ( 1 3600 600 604800 300 )
@ IN NS  ns.42.school.
ns IN A  10.255.0.54
`,
	}

	if errs := svc.ValidateArtifacts([]render.Output{good}); len(errs) != 0 {
		t.Fatalf("valid zone rejected: %v", errs)
	}

	bad := good
	bad.Content = `$ORIGIN 42.school.
$TTL 300
@ IN SOA ns.42.school. hostmaster.42.school. ( 1 3600 600 604800 300 )
www IN A not-an-address
`

	errs := svc.ValidateArtifacts([]render.Output{bad})
	if len(errs) == 0 {
		t.Fatal("expected a broken zone file to be rejected before it is written")
	}

	if errs[0].Tool != artifacterr.ToolNamedCheckzone {
		t.Fatalf("wrong tool attributed: %+v", errs[0])
	}

	if errs[0].Artifact != bad.Name {
		t.Fatalf("wrong artifact attributed: %+v", errs[0])
	}

	if errs[0].Line == 0 {
		t.Fatalf("expected a line number from named-checkzone: %+v", errs[0])
	}
}
