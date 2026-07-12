package main

import (
	"github.com/ChevalRouting/routier/pkg/api/collect"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newCollectStatsCommand() *cobra.Command {
	var dbPath, configPath string
	cmd := &cobra.Command{
		Use:    "collect-stats",
		Short:  "collect interface, BGP, protocol and neighbor stats into the database",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := collect.Run(dbPath, configPath); err != nil {
				return err
			}

			log.Info().Str("db", dbPath).Msg("stats collected")
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "/var/lib/routier/web.db", "path to SQLite database")
	cmd.Flags().StringVar(&configPath, "config", "/etc/routier/config.yml", "path to routier config for collection intervals")
	return cmd
}
