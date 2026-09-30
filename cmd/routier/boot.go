package main

import (
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/host/motd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newBootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boot [config]",
		Short: "apply config on boot without rollback timer",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadAndValidate(configArg(args), true)
			if err != nil {
				return err
			}

			if err := managers.Boot(cmd.Context(), cfg, friends.LoadCacheVars(defaultFriendsCache)); err != nil {
				return err
			}

			motd.Write(cmd.Context(), "", "")
			log.Info().Msg("applied")
			return nil
		},
	}

	return cmd
}
