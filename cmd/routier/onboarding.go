package main

import (
	"fmt"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/spf13/cobra"
)

func newOnboardingCommand() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "onboarding",
		Short: "manage the web setup wizard state",
	}

	cmd.PersistentFlags().StringVar(&dbPath, "db", "/var/lib/routier/web.db", "path to SQLite database")
	cmd.AddCommand(
		newOnboardingCompleteCommand(&dbPath),
		newOnboardingResetCommand(&dbPath),
		newOnboardingStatusCommand(&dbPath),
	)

	return cmd
}

func newOnboardingCompleteCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "complete",
		Short: "mark onboarding complete so the web UI skips the setup wizard",
		Long:  "Mark onboarding complete so the web UI skips the setup wizard. Use this after importing an existing config that is already fully configured.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return setOnboardingComplete(cmd, *dbPath, "true")
		},
	}
}

func newOnboardingResetCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "clear onboarding state so the web UI shows the setup wizard again",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return setOnboardingComplete(cmd, *dbPath, "false")
		},
	}
}

func newOnboardingStatusCommand(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "print whether onboarding is complete",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			db, err := webdb.InitDB(ctx, *dbPath)
			if err != nil {
				return err
			}

			defer db.Close()

			fmt.Fprintln(cmd.OutOrStdout(), webdb.Setting(ctx, db, webdb.SettingOnboardingComplete, "false"))
			return nil
		},
	}
}

func setOnboardingComplete(cmd *cobra.Command, dbPath, value string) error {
	ctx := cmd.Context()
	db, err := webdb.InitDB(ctx, dbPath)
	if err != nil {
		return err
	}

	defer db.Close()

	if err := webdb.SetSetting(ctx, db, webdb.SettingOnboardingComplete, value); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "onboarding_complete=%s\n", value)
	return nil
}
