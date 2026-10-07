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
		RunE: func(cmd *cobra.Command, unusedArg2 []string) error {
			return newPruneStatsCommandCallback(dbPath, cmd, unusedArg2)
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "/var/lib/routier/web.db", "path to SQLite database")
	return cmd
}

func newPruneStatsCommandCallback(dbPath string, cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	db, err := webdb.InitDB(ctx, dbPath)
	if err != nil {
		return err
	}

	defer func(action func() error) { _ = action() }(db.Close)

	if err := webdb.PruneOldStats(ctx, db); err != nil {
		return err
	}

	log.Info().Str("db", dbPath).Msg("stats pruned")
	return nil
}
