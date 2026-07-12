package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(VERSION)
		},
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "routier",
		Short:         "Broom Broom",
		Version:       VERSION,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := validateConfig(validateLogLevel); err != nil {
				return err
			}

			return injectContext(cmd, args)
		},
	}

	rootFlags(root)
	root.AddCommand(
		newVersionCommand(),
		newWriteMOTDCommand(),
		newFirstbootCommand(),
		newTakeoverCommand(),
		newValidateCommand(),
		newMigrateCommand(),
		newPlanCommand(),
		newApplyCommand(),
		newEditCommand(),
		newBootCommand(),
		newConfirmCommand(),
		newRollbackCommand(),
		newInitCommand(),
		newWatchdogCommand(),
		newCollectStatsCommand(),
		newPruneStatsCommand(),
		newAnycastCommand(),
		newSwitchoverCommand(),
		newActivateCommand(),
		newBackupsCommand(),
		newRestoreCommand(),
		newHistoryCommand(),
		newSnapshotsCommand(),
		newMacrosCommand(),
		newFriendsCommand(),
		newNatCommand(),
		newDHCPCommand(),
		newIpcalcCommand(),
	)
	if isLiveISO() {
		root.AddCommand(newSetupCommand())
	}

	return root
}

func rootFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&cli.LogLevel, "log-level", "v", "info", "log level (debug, info, warn, error)")
}
