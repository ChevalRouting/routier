package main

import (
	"github.com/ChevalRouting/routier/pkg/api/collect"
	"github.com/spf13/cobra"
)

func newCollectProbesCommand() *cobra.Command {
	var dbPath, configPath string
	cmd := &cobra.Command{Use: "collect-probes", Short: "run configured ping probes and store their average RTT", Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return collect.RunProbes(cmd.Context(), dbPath, configPath)
		}}
	cmd.Flags().StringVar(&dbPath, "db", "/var/lib/routier/web.db", "path to SQLite database")
	cmd.Flags().StringVar(&configPath, "config", "/etc/routier/config.yml", "path to routier config")
	return cmd
}
