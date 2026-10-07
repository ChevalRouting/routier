package main

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "dump starter config and default templates",
		RunE: func(unusedArg1 *cobra.Command, unusedArg2 []string) error {
			return newInitCommandCallback(dir, unusedArg1, unusedArg2)
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "output directory")
	return cmd
}

func dumpTemplates(base string) error {
	tplDir := filepath.Join(base, "templates")
	return fs.WalkDir(render.Embedded(), "defaults", func(path string, d fs.DirEntry, err error) error { return dumpTemplatesCallback(tplDir, path, d, err) })
}

func dumpStarterConfig(dir string) error {
	return os.WriteFile(filepath.Join(dir, "config.yml"), config.DefaultBytes(), 0644)
}

func newInitCommandCallback(dir string, _ *cobra.Command, _ []string) error {
	if err := dumpTemplates(dir); err != nil {
		return err
	}

	if err := dumpStarterConfig(dir); err != nil {
		return err
	}

	log.Info().Str("dir", dir).Msg("created")
	log.Info().Msg("config.yml: edit this")
	log.Info().Msg("templates/: override any template here")
	return nil
}

func dumpTemplatesCallback(tplDir string, path string, d fs.DirEntry, err error) error {
	if err != nil || d.IsDir() {
		return err
	}

	data, err := render.ReadEmbedded(path)
	if err != nil {
		return err
	}

	rel := path[len("defaults/"):]
	dest := filepath.Join(tplDir, rel)
	_ = os.MkdirAll(filepath.Dir(dest), 0755)
	return os.WriteFile(dest, data, 0644)
}
