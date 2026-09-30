package managers

import (
	"context"
	"fmt"
	"os"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

func persistDDNSKey(cfg *config.Config, configPath string) {
	if configPath == "" {
		return
	}

	info, err := os.Stat(configPath)
	if err != nil || !info.Mode().IsRegular() {
		log.Warn().Str("path", configPath).Msg("ddns: generated TSIG key not persisted (config is not a regular file); set dhcp.ddns.key explicitly")
		return
	}

	if err := config.Save(configPath, cfg); err != nil {
		log.Warn().Err(err).Str("path", configPath).Msg("ddns: failed to persist generated TSIG key")
	}
}

type Result struct {
	SnapID  string
	Warning string
	Armed   bool
}

func Resolve(cfg *config.Config, vars map[string]friends.Vars) (*config.Config, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if !friends.HasTemplate(string(data)) {
		return cfg, nil
	}

	resolved, err := friends.Interpolate(string(data), vars)
	if err != nil {
		return nil, err
	}

	out, err := config.LoadBytes([]byte(resolved))
	if err != nil {
		return nil, err
	}

	out.BaseDir = cfg.BaseDir
	return out, nil
}

func Apply(ctx context.Context, cfg *config.Config, vars map[string]friends.Vars, opts ApplyOptions, armTimeout int) (Result, error) {
	config.ResolveInterfaces(cfg)

	if config.EnsureDDNSKey(cfg) {
		persistDDNSKey(cfg, opts.ConfigPath)
	}

	outputs, err := render.All(cfg, render.WithFriends(vars))
	if err != nil {
		return Result{}, err
	}

	snapID, changed, err := ApplyConfig(ctx, cfg, outputs, opts)
	if err != nil {
		if snapID != "" {
			if _, rollbackErr := Rollback(snapID); rollbackErr != nil {
				return Result{SnapID: snapID}, fmt.Errorf("apply failed: %w; rollback failed: %w", err, rollbackErr)
			}
		}

		return Result{SnapID: snapID}, err
	}

	res := Result{SnapID: snapID}
	if armTimeout > 0 && snapID != "" && len(changed) > 0 {
		if perr := SetPending(snapID, armTimeout, opts.Source); perr != nil {
			res.Warning = fmt.Sprintf("failed to set pending: %v", perr)
		} else {
			res.Armed = true
		}
	}

	return res, nil
}

func CoordinatedApply(ctx context.Context, f *config.Friend, friendPayload PushPayload, persistLocal func() error, localCfg *config.Config, vars map[string]friends.Vars, opts ApplyOptions) error {
	if err := PushApplyConfirm(f, friendPayload); err != nil {
		return fmt.Errorf("apply to friend: %w", err)
	}

	if persistLocal != nil {
		if err := persistLocal(); err != nil {
			return fmt.Errorf("persist local config: %w", err)
		}
	}

	res, err := Apply(ctx, localCfg, vars, opts, WatchdogTimeout)
	if err != nil {
		return fmt.Errorf("local apply: %w", err)
	}

	if !res.Armed {
		return nil
	}

	if !FriendReachable(f) {
		return fmt.Errorf("friend unreachable after local apply; rolling back")
	}

	_, err = ConfirmPending()
	return err
}
