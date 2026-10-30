package managers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/switches"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

var switchImageDir = "/var/lib/routier/switch-images"

type switchTarget struct {
	name     string
	desired  *config.Switch
	previous *config.Switch
}

type switchEngineOpener func(ctx context.Context) (*switches.Engine, func(), error)

var openSwitchEngine switchEngineOpener = defaultSwitchEngine

func defaultSwitchEngine(ctx context.Context) (*switches.Engine, func(), error) {
	return switches.OpenEngine(ctx, switches.DefaultDBPath)
}

func SetSwitchEngineOpener(opener func(ctx context.Context) (*switches.Engine, func(), error)) func() {
	previous := openSwitchEngine
	openSwitchEngine = opener
	return func() { openSwitchEngine = previous }
}

func switchesToReconcile(names []string, cfg *config.Config) []string {
	var out []string
	for _, n := range names {
		name, ok := render.SwitchFromArtifactName(n)
		if ok && cfg.Switches[name] != nil {
			out = append(out, name)
		}
	}

	return out
}

func reconcileSwitches(ctx context.Context, names []string, cfg *config.Config, previous map[string]*config.Switch, snapID string, dryRun, bestEffort bool, warnings *[]string) error {
	var targets []switchTarget
	for _, name := range switchesToReconcile(names, cfg) {
		targets = append(targets, switchTarget{name: name, desired: cfg.Switches[name], previous: previous[name]})
	}

	return runSwitchTargets(ctx, targets, snapID, dryRun, bestEffort, warnings)
}

func restoreSwitches(ctx context.Context, names []string, restored *config.Config, rolledBack map[string]*config.Switch, snapID string) error {
	var targets []switchTarget
	for _, n := range names {
		name, ok := render.SwitchFromArtifactName(n)
		if !ok {
			continue
		}

		desired := switches.RestoreDefinition(restored.Switches[name], rolledBack[name], loadSwitchImage(snapID, name))
		if desired != nil {
			targets = append(targets, switchTarget{name: name, desired: desired, previous: rolledBack[name]})
		}
	}

	return runSwitchTargets(ctx, targets, "", false, false, nil)
}

func runSwitchTargets(ctx context.Context, targets []switchTarget, snapID string, dryRun, bestEffort bool, warnings *[]string) error {
	if len(targets) == 0 {
		return nil
	}

	engine, closeEngine, err := openSwitchEngine(ctx)
	if err != nil {
		return switchFailure(bestEffort, warnings, fmt.Errorf("switches: %w", err))
	}

	defer closeEngine()

	var errs []error
	for _, target := range targets {
		count, err := engine.Reconcile(ctx, target.name, target.desired, target.previous, dryRun, logSwitchStep, imageSaver(snapID, target.name))
		if err != nil {
			errs = append(errs, fmt.Errorf("switch %s: %w", target.name, err))
			continue
		}

		log.Info().Str("switch", target.name).Int("steps", count).Bool("dry_run", dryRun).Msg("switch reconciled")
	}

	if len(errs) == 0 {
		return nil
	}

	return switchFailure(bestEffort, warnings, errors.Join(errs...))
}

func switchFailure(bestEffort bool, warnings *[]string, err error) error {
	if !bestEffort {
		return err
	}

	if warnings != nil {
		*warnings = append(*warnings, strings.Split(err.Error(), "\n")...)
	}

	log.Warn().Err(err).Msg("switch reconcile skipped")
	return nil
}

func logSwitchStep(format string, args ...any) {
	log.Info().Msg(fmt.Sprintf(format, args...))
}

func switchImagePath(snapID, name string) string {
	return filepath.Join(switchImageDir, snapID, name+".json")
}

func imageSaver(snapID, name string) switches.BeforeChange {
	if snapID == "" {
		return nil
	}

	return func(live types.SwitchState) error { return saveSwitchImage(snapID, name, live) }
}

func saveSwitchImage(snapID, name string, live types.SwitchState) error {
	path := switchImagePath(snapID, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.Marshal(live)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func loadSwitchImage(snapID, name string) *types.SwitchState {
	data, err := os.ReadFile(switchImagePath(snapID, name))
	if err != nil {
		return nil
	}

	var state types.SwitchState
	if json.Unmarshal(data, &state) != nil {
		return nil
	}

	return &state
}

func pruneSwitchImages() {
	entries, err := os.ReadDir(switchImageDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		snapshot := filepath.Join("/var/lib/routier/snapshots", entry.Name()+".json")
		if _, err := os.Stat(snapshot); os.IsNotExist(err) {
			_ = os.RemoveAll(filepath.Join(switchImageDir, entry.Name()))
		}
	}
}
