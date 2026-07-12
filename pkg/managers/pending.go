package managers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/applylog"
	"github.com/ChevalRouting/routier/pkg/config"
	anyk "github.com/m-vinc/anyk"
	"github.com/rs/zerolog/log"
)

type pendingState struct {
	SnapID  string    `json:"snap_id"`
	Created time.Time `json:"created"`
	Timeout int       `json:"timeout"`
	Source  string    `json:"source,omitempty"`
}

const WatchdogTimeout = 60

type PendingApply struct {
	SnapID    string `json:"snap_id"`
	Timeout   int    `json:"timeout"`
	Remaining int    `json:"remaining"`
}

func PendingStatus() (*PendingApply, error) {
	data, err := os.ReadFile(pendingFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var state pendingState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	remaining := state.Timeout - int(time.Since(state.Created).Seconds())
	if remaining < 0 {
		remaining = 0
	}

	return &PendingApply{SnapID: state.SnapID, Timeout: state.Timeout, Remaining: remaining}, nil
}

func SetPending(snapID string, timeout int, source string) error {
	_ = os.MkdirAll(filepath.Dir(pendingFile), 0700)

	data, err := json.Marshal(pendingState{
		SnapID:  snapID,
		Created: time.Now(),
		Timeout: timeout,
		Source:  source,
	})
	if err != nil {
		return err
	}

	return os.WriteFile(pendingFile, data, 0600)
}

func takePending() (snapID, source string, err error) {
	data, err := os.ReadFile(pendingFile)
	if err != nil {
		return "", "", fmt.Errorf("no pending apply")
	}

	_ = os.Remove(pendingFile)
	var state pendingState
	if err := json.Unmarshal(data, &state); err == nil {
		return state.SnapID, state.Source, nil
	}

	return string(data), "", nil
}

func ClearPending() (string, error) {
	id, _, err := takePending()
	return id, err
}

func ConfirmPending() (string, error) {
	id, source, err := takePending()
	if err != nil {
		return "", err
	}

	if id != "" {
		applylog.MarkConfirmed(id)
	}

	if !strings.HasPrefix(source, "macro:") {
		runAnycast()
	}

	return id, nil
}

func runAnycast() {
	cfg, err := config.Load(LastAppliedPath)
	if err != nil ||
		cfg.Routing == nil ||
		cfg.Routing.Anycast == nil ||
		len(cfg.Routing.Anycast.Services) == 0 ||
		cfg.Routing.BGP == nil {
		return
	}

	ctx := log.Logger.WithContext(context.Background())
	if err := anyk.Run(ctx, cfg.Routing.BGP.ASN, toAnykServices(cfg.Routing.Anycast.Services)); err != nil {
		log.Warn().Err(err).Msg("anycast run failed, next cron cycle will retry")
	}
}

func RunWatchdog() error {
	data, err := os.ReadFile(pendingFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Msg("watchdog: no pending apply")
			return nil
		}

		return err
	}

	var state pendingState
	if err := json.Unmarshal(data, &state); err != nil {
		snapID := strings.TrimSpace(string(data))
		info, serr := os.Stat(pendingFile)
		if serr != nil {
			return nil
		}

		if time.Since(info.ModTime()) > 60*time.Second {
			log.Warn().Str("snapshot", snapID).Msg("watchdog: timeout, rolling back")
			_, err := RollbackSource(snapID, "watchdog")
			return err
		}

		return nil
	}

	log.Debug().Str("snapshot", state.SnapID).Int("timeout", state.Timeout).Msg("watchdog: pending apply")
	if time.Since(state.Created) > time.Duration(state.Timeout)*time.Second {
		log.Warn().Str("snapshot", state.SnapID).Msg("watchdog: timeout, rolling back")
		_, err := RollbackSource(state.SnapID, "watchdog")
		return err
	}

	return nil
}

var pendingFile = "/var/lib/routier/pending"

func SetPendingFile(p string) func() {
	prev := pendingFile
	pendingFile = p
	return func() { pendingFile = prev }
}
