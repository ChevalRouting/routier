package managers

import (
	"context"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
)

func Boot(ctx context.Context, cfg *config.Config, vars map[string]friends.Vars) error {
	_, _ = ClearPending()

	outputs, err := render.All(cfg, render.WithFriends(vars))
	if err != nil {
		return err
	}

	_, _, err = ApplyConfig(ctx, cfg, outputs, ApplyOptions{
		ReloadAll: true,
		Source:    "boot",
	})
	return err
}
