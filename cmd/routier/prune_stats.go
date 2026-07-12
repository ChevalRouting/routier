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
		RunE: func(_ *cobra.Command, _ []string) error {
			db, err := webdb.InitDB(dbPath)
			if err != nil {
				return err
			}

			defer db.Close()

			webdb.PruneOldStats(db)
			log.Info().Str("db", dbPath).Msg("stats pruned")
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "/var/lib/routier/web.db", "path to SQLite database")
	return cmd
}
