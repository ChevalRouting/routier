package workers

import (
	"context"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/rs/zerolog/log"
)

func prune(ctx context.Context, db *webdb.DB) {
	if err := webdb.PruneOldStats(ctx, db); err != nil {
		log.Warn().Err(err).Msg("prune old stats")
	}
}

func StartCleanup(ctx context.Context, db *webdb.DB) {
	go func() { startCleanupCallback(ctx, db) }()
}

func startCleanupCallback(ctx context.Context, db *webdb.DB) {
	prune(ctx, db)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			prune(ctx, db)
		}
	}
}
