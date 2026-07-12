package main

import (
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/spf13/cobra"
)

func newWatchdogCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "watchdog",
		Short:  "check pending rollback timeout (called by init/cron)",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return managers.RunWatchdog()
		},
	}
}
