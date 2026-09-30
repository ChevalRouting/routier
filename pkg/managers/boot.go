package managers

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/rs/zerolog/log"
)

func Boot(ctx context.Context, cfg *config.Config, vars map[string]friends.Vars) error {
	outputs, err := render.All(cfg, render.WithFriends(vars))
	if err != nil {
		return err
	}

	skipWG := hasSwitchoverVRRP(cfg)
	if skipWG {
		log.Info().Msg("VRRP switchover active: skipping WireGuard init on boot")
	}

	_, _, err = ApplyConfig(ctx, cfg, outputs, ApplyOptions{
		ReloadAll:     true,
		SkipWireguard: skipWG,
		Source:        "boot",
	})
	return err
}

func hasSwitchoverVRRP(cfg *config.Config) bool {
	if cfg.HA == nil {
		return false
	}

	for _, inst := range cfg.HA.VRRP {
		if inst.Switchover {
			return true
		}
	}

	return false
}
