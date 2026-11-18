package managers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ChevalRouting/routier/pkg/apply"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/state/applylog"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/rs/zerolog/log"
)

func Rollback(id string) (string, error) {
	return RollbackSource(id, "rollback")
}

func RollbackSource(id, source string) (resolvedID string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	release, err := acquireApplyLock(ctx)
	if err != nil {
		return "", err
	}

	defer release()

	if source == "watchdog" {
		state, perr := readPending()
		if perr != nil || state.SnapID != id {
			log.Info().Str("snapshot", id).Msg("watchdog: apply confirmed or replaced meanwhile, skipping rollback")
			return "", nil
		}
	}

	rec := applylog.Start(source, "")
	defer func() { finishRollback(resolvedID, err, rec) }()

	if id == "" {
		if data, err := os.ReadFile(pendingFile); err == nil {
			var state pendingState
			if err := json.Unmarshal(data, &state); err == nil {
				id = state.SnapID
			} else {
				id = string(data)
			}
		}
	}

	rolledBackSwitches := render.ReadSwitchArtifacts()
	names, restoreErr := apply.Rollback(id)
	if restoreErr != nil && len(names) == 0 {
		return "", restoreErr
	}

	var restoredCfg *config.Config
	if c, lerr := loadRollbackConfig(LastAppliedPath); lerr == nil {
		restoredCfg = c
	} else if c, lerr := loadRollbackConfig("/etc/routier/config.yml"); lerr == nil {
		restoredCfg = c
	} else {
		return id, fmt.Errorf("rollback: could not load restored config for netlink reconcile: %w", lerr)
	}

	if err = svc.ReloadFromOutputs(names, restoredCfg, false); err != nil {
		return id, err
	}

	if serr := restoreSwitches(ctx, names, restoredCfg, rolledBackSwitches, id); serr != nil {
		log.Error().Err(serr).Msg("rollback: switches could not be restored, check them by hand")
	}

	if restoreErr != nil {
		return id, fmt.Errorf("rollback restored files only partially: %w", restoreErr)
	}

	_ = os.Remove(pendingFile)
	return id, nil
}

func loadRollbackConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}

	config.ResolveInterfaces(cfg)
	for name, iface := range cfg.Interfaces {
		if iface.Device == "" {
			return nil, fmt.Errorf("interface %s: device could not be resolved", name)
		}
	}

	return cfg, nil
}

func finishRollback(resolvedID string, err error, rec *applylog.Recorder) {
	result := "rolledback"
	if err != nil {
		result = "failed"
	}

	if err == nil {
		applylog.MarkRolledBack(resolvedID)
	}

	rec.Finish(resolvedID, result)
}
