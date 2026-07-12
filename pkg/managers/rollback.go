package managers

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/ChevalRouting/routier/pkg/apply"
	"github.com/ChevalRouting/routier/pkg/applylog"
	"github.com/ChevalRouting/routier/pkg/config"
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

	_ = os.Remove(pendingFile)

	var restoredCfg *config.Config
	if c, lerr := config.Load(LastAppliedPath); lerr == nil {
		restoredCfg = c
	} else if c, lerr := config.Load("/etc/routier/config.yml"); lerr == nil {
		restoredCfg = c
	} else {
		log.Warn().Err(lerr).Msg("rollback: could not load restored config for netlink reconcile")
	}

	err = svc.ReloadFromOutputs(names, restoredCfg, false, false)
	return id, err
}
