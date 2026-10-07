package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "migrate [config]",
		Short: "migrate config file(s) to the current schema version",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(unusedArg1 *cobra.Command, args []string) error {
			return newMigrateCommandCallback(dryRun, unusedArg1, args)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would change without writing")
	return cmd
}

func newMigrateCommandCallback(dryRun bool, _ *cobra.Command, args []string) error {
	path := configArg(args)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	var files []string
	if info.IsDir() {
		for _, ext := range []string{"*.yml", "*.yaml"} {
			found, _ := filepath.Glob(filepath.Join(path, ext))
			files = append(files, found...)
		}

		sort.Strings(files)
	} else {
		files = []string{path}
	}

	migrated := false
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return err
		}

		out, changed, err := config.MigrateBytes(data)
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}

		if !changed {
			continue
		}

		migrated = true
		if dryRun {
			log.Info().Str("file", f).Msgf("would migrate to %s", config.CurrentVersion)
			continue
		}

		mode := os.FileMode(0600)
		if fi, err := os.Stat(f); err == nil {
			mode = fi.Mode().Perm()
		}

		if err := os.WriteFile(f, out, mode); err != nil {
			return err
		}

		log.Info().Str("file", f).Msgf("migrated to %s", config.CurrentVersion)
	}

	if !migrated {
		log.Info().Str("version", config.CurrentVersion).Msg("config already up to date")
	}

	return nil
}
