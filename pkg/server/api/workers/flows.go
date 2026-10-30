package workers

import (
	"context"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/telemetry/flows"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const (
	flowsConfigInterval = 10 * time.Second
	flowsRetryDelay     = 30 * time.Second
)

type flowStore struct {
	db *webdb.DB
}

func (s flowStore) InsertFlows(ctx context.Context, rows []types.Flow) error {
	return webdb.InsertFlows(ctx, s.db, rows)
}

func (s flowStore) SaveFlowName(ctx context.Context, address, name string, resolvedAt int64) error {
	return webdb.SaveFlowName(ctx, s.db, address, name, resolvedAt)
}

func (s flowStore) PruneFlows(ctx context.Context, before int64, keep int) error {
	return webdb.PruneFlows(ctx, s.db, before, keep)
}

type flowRunner struct {
	store     flowStore
	settings  flows.Settings
	cancel    context.CancelFunc
	done      chan struct{}
	startedAt time.Time
}

func StartFlows(ctx context.Context, g *Group, db *webdb.DB, configPath string) {
	r := &flowRunner{store: flowStore{db: db}}
	g.Go(func() { r.run(ctx, configPath) })
}

func (r *flowRunner) run(ctx context.Context, configPath string) {
	ticker := time.NewTicker(flowsConfigInterval)
	defer ticker.Stop()

	for {
		r.reconcile(ctx, configPath)
		select {
		case <-ctx.Done():
			r.stop()
			return
		case <-ticker.C:
		}
	}
}

func (r *flowRunner) running() bool {
	if r.done == nil {
		return false
	}

	select {
	case <-r.done:
		r.cancel()
		r.done = nil
		return false
	default:
		return true
	}
}

func (r *flowRunner) reconcile(ctx context.Context, configPath string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return
	}

	config.ResolveInterfaces(cfg)
	settings, enabled := flows.SettingsFromConfig(cfg)
	running := r.running()

	switch {
	case !enabled:
		r.stop()
	case running && settings.Equal(r.settings):
	case !running && settings.Equal(r.settings) && time.Since(r.startedAt) < flowsRetryDelay:
	default:
		r.stop()
		r.start(ctx, settings)
	}
}

func (r *flowRunner) start(ctx context.Context, settings flows.Settings) {
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	r.settings, r.cancel, r.done = settings, cancel, done

	go r.runAgent(runCtx, settings, done)

	r.startedAt = time.Now()
}

func (r *flowRunner) runAgent(ctx context.Context, settings flows.Settings, done chan<- struct{}) {
	defer close(done)
	if err := flows.Run(ctx, settings, r.store); err != nil {
		log.Error().Err(err).Msg("flows: agent stopped")
	}
}

func (r *flowRunner) stop() {
	if r.done == nil {
		return
	}

	r.cancel()
	<-r.done
	r.done = nil
}
