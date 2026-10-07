package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/host/boot"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(_ *cobra.Command, _ []string) {
			_, _ = fmt.Println(VERSION)
		},
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:               "routier",
		Short:             "Broom Broom",
		Version:           VERSION,
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: newRootCommandHandler,
	}

	rootFlags(root)
	root.AddCommand(
		newVersionCommand(),
		newWriteMOTDCommand(),
		newIssueCommand(),
		newFirstbootCommand(),
		newBootHookCommand(),
		newOnboardingCommand(),
		newUpdateCommand(),
		newUpgradeCommand(),
		newTakeoverCommand(),
		newValidateCommand(),
		newMigrateCommand(),
		newPlanCommand(),
		newDiffCommand(),
		newApplyCommand(),
		newEditCommand(),
		newBootCommand(),
		newConfirmCommand(),
		newRollbackCommand(),
		newInitCommand(),
		newWatchdogCommand(),
		newCollectStatsCommand(),
		newCollectProbesCommand(),
		newPruneStatsCommand(),
		newAnycastCommand(),
		newSwitchoverCommand(),
		newActivateCommand(),
		newBackupsCommand(),
		newRestoreCommand(),
		newHistoryCommand(),
		newSnapshotsCommand(),
		newFailuresCommand(),
		newMacrosCommand(),
		newFriendsCommand(),
		newNatCommand(),
		newDHCPCommand(),
		newDNSCommand(),
		newIpcalcCommand(),
	)
	if boot.IsLive() {
		root.AddCommand(newSetupCommand())
	}

	return root
}

func rootFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&cli.LogLevel, "log-level", "v", "info", "log level (debug, info, warn, error)")
}

func newRootCommandHandler(cmd *cobra.Command, args []string) error {
	if err := validateConfig(validateLogLevel); err != nil {
		return err
	}

	return injectContext(cmd, args)
}
