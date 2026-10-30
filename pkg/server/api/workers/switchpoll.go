package workers

import (
	"context"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/server/api/switchcache"
	"github.com/ChevalRouting/routier/pkg/switches"
)

const defaultSwitchPollInterval = 60

func StartSwitchPoll(ctx context.Context, g *Group, engine *switches.Engine, configPath string) {
	g.Go(func() {
		every(ctx, func() time.Duration { return switchPollInterval(configPath) }, func() { switchcache.Poll(ctx, engine, configPath) })
	})
}

func switchPollInterval(configPath string) time.Duration {
	seconds := defaultSwitchPollInterval
	if cfg, err := config.Load(configPath); err == nil && cfg.Monitoring != nil && cfg.Monitoring.Collection.Switches > 0 {
		seconds = cfg.Monitoring.Collection.Switches
	}

	return time.Duration(seconds) * time.Second
}
