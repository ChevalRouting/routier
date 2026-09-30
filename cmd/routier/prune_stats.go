package main

import (
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newPruneStatsCommand() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:    "prune-stats",
		Short:  "delete stats history past the retention window and reclaim disk space",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			db, err := webdb.InitDB(ctx, dbPath)
			if err != nil {
				return err
			}

			defer db.Close()

			webdb.PruneOldStats(ctx, db)
			log.Info().Str("db", dbPath).Msg("stats pruned")
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "/var/lib/routier/web.db", "path to SQLite database")
	return cmd
}
