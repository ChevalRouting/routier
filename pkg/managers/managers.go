package managers

import (
	"context"
	"fmt"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
	"gopkg.in/yaml.v3"
)

type Result struct {
	SnapID  string
	Warning string
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

	outputs, err := render.All(cfg, render.WithFriends(vars))
	if err != nil {
		return Result{}, err
	}

	snapID, err := ApplyConfig(ctx, cfg, outputs, opts)
	if err != nil {
		if snapID != "" {
			_, _ = Rollback(snapID)
		}

		return Result{SnapID: snapID}, err
	}

	res := Result{SnapID: snapID}
	if armTimeout > 0 && snapID != "" {
		if perr := SetPending(snapID, armTimeout, opts.Source); perr != nil {
			res.Warning = fmt.Sprintf("failed to set pending: %v", perr)
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

	if _, err := Apply(ctx, localCfg, vars, opts, WatchdogTimeout); err != nil {
		return fmt.Errorf("local apply: %w", err)
	}

	if !FriendReachable(f) {
		return fmt.Errorf("friend unreachable after local apply; rolling back")
	}

	_, err := ConfirmPending()
	return err
}
