package managers

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	raw, err := os.ReadFile(configPath)
	if err == nil {
		raw, err = withDDNSKey(raw, cfg)
	}

	if err == nil {
		err = writeConfigFile(configPath, raw)
	}

	if err != nil {
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
	if !opts.DryRun {
		if err := RequireNoPending(); err != nil {
			return Result{}, err
		}
	}

	persist := opts.PersistConfig

	config.ResolveInterfaces(cfg)

	if config.EnsureDDNSKey(cfg) {
		if persist == nil {
			persistDDNSKey(cfg, opts.ConfigPath)
		} else if carried, err := withDDNSKey(persist, cfg); err != nil {
			return Result{}, err
		} else {
			persist = carried
		}
	}

	outputs, err := render.All(cfg, render.WithFriends(vars))
	if err != nil {
		return Result{}, err
	}

	var warnings []string
	arm := armRequest{exclusive: !opts.DryRun, timeout: armTimeout, persist: persist}
	snapID, _, armed, err := applyConfig(ctx, cfg, outputs, opts, arm, &warnings)
	if err != nil {
		if snapID != "" {
			if _, rollbackErr := Rollback(snapID); rollbackErr != nil {
				return Result{SnapID: snapID}, fmt.Errorf("apply failed: %w; rollback failed: %w", err, rollbackErr)
			}
		}

		return Result{SnapID: snapID}, err
	}

	res := Result{SnapID: snapID, Armed: armed}
	if !res.Armed && persist != nil && !opts.DryRun {
		if err := writeConfigFile(opts.configPath(), persist); err != nil {
			return res, fmt.Errorf("applied but failed to persist config: %w", err)
		}
	}

	res.Warning = strings.Join(warnings, "; ")
	return res, nil
}

func CoordinatedApply(ctx context.Context, f *config.Friend, friendPayload PushPayload, localCfg *config.Config, vars map[string]friends.Vars, opts ApplyOptions) (string, error) {
	if err := RequireNoPending(); err != nil {
		return "", err
	}

	friendWarning, err := PushApplyConfirm(f, friendPayload)
	if err != nil {
		return "", fmt.Errorf("apply to friend: %w", err)
	}

	res, err := Apply(ctx, localCfg, vars, opts, WatchdogTimeout)
	if err != nil {
		return "", fmt.Errorf("local apply: %w", err)
	}

	warning := combineWarnings(f.Name, friendWarning, res.Warning)

	if !res.Armed {
		return warning, nil
	}

	if !FriendReachable(f) {
		return warning, fmt.Errorf("friend unreachable after local apply; rolling back")
	}

	if _, err = ConfirmPending(); err != nil {
		return warning, err
	}

	return warning, nil
}

func combineWarnings(friendName, friendWarning, localWarning string) string {
	var parts []string
	if friendWarning != "" {
		parts = append(parts, fmt.Sprintf("friend %s: %s", friendName, friendWarning))
	}

	if localWarning != "" {
		parts = append(parts, localWarning)
	}

	return strings.Join(parts, "; ")
}
