package managers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ChevalRouting/routier/pkg/apply"
	"github.com/ChevalRouting/routier/pkg/state/applylog"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/svc"
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

	rec := applylog.Start(source, "")
	defer func() {
		result := "rolledback"
		if err != nil {
			result = "failed"
		}

		rec.Finish(resolvedID, result)
		if err == nil {
			applylog.MarkRolledBack(resolvedID)
		}
	}()

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

	names, err := apply.Rollback(id)
	if err != nil {
		return "", err
	}

	var restoredCfg *config.Config
	if c, lerr := loadRollbackConfig(LastAppliedPath); lerr == nil {
		restoredCfg = c
	} else if c, lerr := loadRollbackConfig("/etc/routier/config.yml"); lerr == nil {
		restoredCfg = c
	} else {
		return id, fmt.Errorf("rollback: could not load restored config for netlink reconcile: %w", lerr)
	}

	if err = svc.ReloadFromOutputs(names, restoredCfg, false, false); err != nil {
		return id, err
	}
	_ = os.Remove(pendingFile)
	return id, nil
}

// Resolved devices and bridge/bond members are runtime-only fields, omitted
// from snapshots. Rebuild them before reconciling the restored network.
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
