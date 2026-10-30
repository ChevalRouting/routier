package workers

import (
	"context"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/rs/zerolog/log"
)

func StartCleanup(ctx context.Context, g *Group, db *webdb.DB) {
	g.Go(func() { every(ctx, hourly, func() { pruneStats(ctx, db) }) })
}

func hourly() time.Duration { return time.Hour }

func pruneStats(ctx context.Context, db *webdb.DB) {
	if err := webdb.PruneOldStats(ctx, db); err != nil {
		log.Warn().Err(err).Msg("prune old stats")
	}
}
