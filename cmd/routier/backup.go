package main

import (
	"fmt"
	"time"

	"github.com/ChevalRouting/routier/pkg/backup"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newBackupsCommand() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:     "backups [config]",
		Aliases: []string{"export"},
		Short:   "create a backup archive of config and referenced files",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfgPath := configArg(args)
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			if output == "" {
				output = fmt.Sprintf("routier-backup-%s.tar.zst", time.Now().Format("20060102-150405"))
			}

			return backup.Create(output, cfgPath, cfg)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "output path (default: routier-backup-TIMESTAMP.tar.zst)")
	return cmd
}

func newRestoreCommand() *cobra.Command {
	var (
		timeout int
		noArm   bool
	)
	cmd := &cobra.Command{
		Use:   "restore <archive>",
		Short: "restore config from backup archive and apply",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := backup.Restore(args[0])
			if err != nil {
				return err
			}

			cfg, err := config.LoadAndValidate(cfgPath, true)
			if err != nil {
				return err
			}

			armTimeout := 0
			if !noArm {
				armTimeout = timeout
			}

			res, err := managers.Apply(cmd.Context(), cfg, friends.LoadCacheVars(defaultFriendsCache),
				managers.ApplyOptions{Source: "restore", ConfigPath: cfgPath}, armTimeout)
			if err != nil {
				return err
			}

			if res.Warning != "" {
				log.Warn().Msg(res.Warning)
			}

			if res.SnapID != "" && !noArm {
				log.Info().Int("timeout", timeout).Msg("confirm pending: routier confirm")
			}

			log.Info().Msg("restored and applied")
			return nil
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 60, "rollback timeout seconds")
	cmd.Flags().BoolVar(&noArm, "no-confirm", false, "skip rollback timer")
	return cmd
}
