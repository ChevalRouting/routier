package main

import (
	"fmt"
	"os"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "migrate [config]",
		Short: "migrate the config file to the current schema version",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runMigrate(dryRun, args)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would change without writing")
	return cmd
}

func runMigrate(dryRun bool, args []string) error {
	path := configArg(args)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	out, changed, err := config.MigrateBytes(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	if !changed {
		log.Info().Str("version", config.CurrentVersion).Msg("config already up to date")
		return nil
	}

	if dryRun {
		log.Info().Str("file", path).Msgf("would migrate to %s", config.CurrentVersion)
		return nil
	}

	mode := os.FileMode(0600)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}

	if err := os.WriteFile(path, out, mode); err != nil {
		return err
	}

	log.Info().Str("file", path).Msgf("migrated to %s", config.CurrentVersion)
	return nil
}
