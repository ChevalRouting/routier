package managers

import (
	"context"
	"errors"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/switches"
)

func failingOpener(context.Context) (*switches.Engine, func(), error) {
	return nil, nil, errors.New("database unavailable")
}

func TestReconcileSwitchesOnlyTouchesChangedSwitches(t *testing.T) {
	cfg := &config.Config{Switches: map[string]*config.Switch{"crs310": {Model: "mikrotik-crs310-8g-2s"}}}
	got := switchesToReconcile([]string{"frr/frr.conf", "switches/crs310.json", "switches/gone.json", "switches/notes.txt"}, cfg)
	if len(got) != 1 || got[0] != "crs310" {
		t.Fatalf("unexpected targets %v", got)
	}
}

func TestReconcileSwitchesFailureModes(t *testing.T) {
	restore := SetSwitchEngineOpener(failingOpener)
	defer restore()

	cfg := &config.Config{Switches: map[string]*config.Switch{"crs310": {Model: "mikrotik-crs310-8g-2s"}}}
	names := []string{"switches/crs310.json"}
	if err := reconcileSwitches(context.Background(), names, cfg, nil, "", false, false, nil); err == nil {
		t.Fatal("apply ignored a switch failure")
	}

	var warnings []string
	if err := reconcileSwitches(context.Background(), names, cfg, nil, "", false, true, &warnings); err != nil || len(warnings) != 1 {
		t.Fatalf("boot did not degrade to a warning: err=%v warnings=%v", err, warnings)
	}

	if err := reconcileSwitches(context.Background(), []string{"frr/frr.conf"}, cfg, nil, "", false, false, nil); err != nil {
		t.Fatalf("engine opened without switch changes: %v", err)
	}
}
