package managers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/host/takeover"
	"github.com/ChevalRouting/routier/pkg/state/applylog"
	anyk "github.com/m-vinc/anyk"
	"github.com/rs/zerolog/log"
)

type pendingState struct {
	SnapID     string    `json:"snap_id"`
	Created    time.Time `json:"created"`
	Timeout    int       `json:"timeout"`
	Source     string    `json:"source,omitempty"`
	ConfigPath string    `json:"config_path,omitempty"`
	Config     []byte    `json:"config,omitempty"`
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
	return setPending(snapID, timeout, source, "", nil)
}

func setPending(snapID string, timeout int, source, configPath string, persist []byte) error {
	_ = os.MkdirAll(filepath.Dir(pendingFile), 0700)

	state := pendingState{
		SnapID:  snapID,
		Created: time.Now(),
		Timeout: timeout,
		Source:  source,
	}
	if persist != nil {
		state.ConfigPath = configPath
		state.Config = persist
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(pendingFile, data, 0600)
}

var ErrNoPending = errors.New("no pending apply")

func takePending() (pendingState, error) {
	state, err := readPending()
	if err != nil {
		return state, err
	}

	_ = os.Remove(pendingFile)

	return state, nil
}

func readPending() (pendingState, error) {
	data, err := os.ReadFile(pendingFile)
	if err != nil {
		return pendingState{}, ErrNoPending
	}

	var state pendingState
	if err := json.Unmarshal(data, &state); err == nil {
		return state, nil
	}

	return pendingState{SnapID: string(data)}, nil
}

func PendingConfig(configPath string) ([]byte, bool) {
	data, err := os.ReadFile(pendingFile)
	if err != nil {
		return nil, false
	}

	var state pendingState
	if err := json.Unmarshal(data, &state); err != nil || state.Config == nil || state.ConfigPath != configPath {
		return nil, false
	}

	return state.Config, true
}

func persistPending(state pendingState) error {
	if state.Config == nil || state.ConfigPath == "" {
		return nil
	}

	if err := writeConfigFile(state.ConfigPath, state.Config); err != nil {
		return fmt.Errorf("confirmed but failed to persist config: %w", err)
	}

	return nil
}

func ClearPending() (string, error) {
	state, err := takePending()
	return state.SnapID, err
}

func ConfirmPending() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	release, err := acquireApplyLock(ctx)
	if err != nil {
		return "", err
	}

	state, err := readPending()
	if err != nil {
		release()
		return "", err
	}

	id, source := state.SnapID, state.Source
	if err := persistPending(state); err != nil {
		release()
		return id, err
	}

	_ = os.Remove(pendingFile)
	release()

	if id != "" {
		applylog.MarkConfirmed(id)
	}

	takeover.EnsureOnConfirm()

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
