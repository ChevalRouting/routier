package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newVRRPNotifyCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:    "vrrp-notify TYPE NAME STATE [PRIORITY]",
		Short:  "reconcile the VRRP transition hooks after a keepalived state change",
		Args:   cobra.RangeArgs(3, 4),
		Hidden: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return runVRRPNotify(configPath, args)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath, "path to config file")
	return cmd
}

func runVRRPNotify(configPath string, args []string) error {
	if args[0] != "INSTANCE" {
		return nil
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log.Info().Str("instance", args[1]).Str("state", args[2]).Msg("vrrp: keepalived reported a state change")
	return svc.SyncVRRPHooks(cfg)
}
