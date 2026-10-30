package switches

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/switches/driver"
	"github.com/ChevalRouting/routier/pkg/switches/model"
	"github.com/ChevalRouting/routier/pkg/types"
)

type recordingStep struct {
	text string
	ran  *[]string
	fail bool
}

func (s recordingStep) Describe() string {
	return s.text
}

func (s recordingStep) Run(context.Context) error {
	if s.fail {
		return errors.New("device refused")
	}

	*s.ran = append(*s.ran, s.text)
	return nil
}

type fakeDriver struct {
	target driver.Target
	ran    *[]string
	failAt int
}

func (f *fakeDriver) State(context.Context) (types.SwitchState, error) {
	return liveState(), nil
}

func (f *fakeDriver) Plan(_ context.Context, plan types.SwitchPlan) ([]driver.Step, error) {
	steps := make([]driver.Step, 0, len(plan.Changes))
	for index, change := range plan.Changes {
		steps = append(steps, recordingStep{text: fmt.Sprintf("%s %v %s", change.Kind, change.Ports, change.Bond), ran: f.ran, fail: index+1 == f.failAt})
	}

	return steps, nil
}

type fakeFleet struct {
	last   *fakeDriver
	ran    []string
	failAt int
}

func (f *fakeFleet) factory(target driver.Target) (driver.Driver, error) {
	f.last = &fakeDriver{target: target, ran: &f.ran, failAt: f.failAt}
	return f.last, nil
}

func testEngine(t *testing.T, fleet *fakeFleet) *Engine {
	t.Helper()

	creds, _, _ := openTestCredentials(t)
	if err := creds.Set(context.Background(), "crs310", map[string]string{"username": "cibi", "password": "s3cret"}); err != nil {
		t.Fatal(err)
	}

	catalog, err := model.Load("")
	if err != nil {
		t.Fatal(err)
	}

	return NewEngine(creds, catalog, driver.Registry{"routeros-rest": fleet.factory})
}

func desiredSwitch() *config.Switch {
	return &config.Switch{
		Model: "mikrotik-crs310-8g-2s", Host: "10.0.0.2", Port: 8443, TLSSkipVerify: true,
		Ports: map[string]*config.SwitchPortConfig{"ether1": access(30), "ether3": access(40)},
	}
}

func discard(string, ...any) {}

func TestEngineReconcileRunsSteps(t *testing.T) {
	fleet := &fakeFleet{}
	engine := testEngine(t, fleet)
	count, err := engine.Reconcile(context.Background(), "crs310", desiredSwitch(), nil, false, discard, nil)
	if err != nil || count != 2 || len(fleet.ran) != 2 {
		t.Fatalf("unexpected result count=%d ran=%v err=%v", count, fleet.ran, err)
	}

	target := fleet.last.target
	if target.Host != "10.0.0.2" || target.Port != 8443 || !target.TLSSkipVerify || target.Credentials["password"] != "s3cret" {
		t.Fatalf("unexpected target %+v", target)
	}
}

func TestEngineDryRunDoesNotTouchTheSwitch(t *testing.T) {
	fleet := &fakeFleet{}
	engine := testEngine(t, fleet)
	var logged []string
	logf := func(format string, args ...any) { logged = append(logged, fmt.Sprintf(format, args...)) }
	count, err := engine.Reconcile(context.Background(), "crs310", desiredSwitch(), nil, true, logf, nil)
	if err != nil || count != 2 || len(fleet.ran) != 0 || len(logged) != 2 {
		t.Fatalf("dry run touched the switch: count=%d ran=%v logged=%v err=%v", count, fleet.ran, logged, err)
	}
}

func TestEngineStopsAtTheFailingStep(t *testing.T) {
	fleet := &fakeFleet{failAt: 2}
	engine := testEngine(t, fleet)
	count, err := engine.Reconcile(context.Background(), "crs310", desiredSwitch(), nil, false, discard, nil)
	if err == nil || count != 1 || len(fleet.ran) != 1 {
		t.Fatalf("unexpected result count=%d ran=%v err=%v", count, fleet.ran, err)
	}
}

func TestEngineSavesTheStateBeforeChanging(t *testing.T) {
	fleet := &fakeFleet{}
	engine := testEngine(t, fleet)
	var saved []types.SwitchState
	before := func(live types.SwitchState) error {
		saved = append(saved, live)
		return nil
	}

	if _, err := engine.Reconcile(context.Background(), "crs310", desiredSwitch(), nil, true, discard, before); err != nil || len(saved) != 0 {
		t.Fatalf("dry run saved a before-image: %v %v", saved, err)
	}

	if _, err := engine.Reconcile(context.Background(), "crs310", desiredSwitch(), nil, false, discard, before); err != nil || len(saved) != 1 || len(saved[0].Ports) == 0 {
		t.Fatalf("before-image not saved: %v %v", saved, err)
	}
}

func TestEngineNeedsCredentials(t *testing.T) {
	engine := testEngine(t, &fakeFleet{})
	if _, err := engine.Reconcile(context.Background(), "crs309", desiredSwitch(), nil, false, discard, nil); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("unexpected error %v", err)
	}
}
