package main

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/spf13/cobra"
)

func newVRRPSyncCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:    "vrrp-sync",
		Short:  "reconcile the VRRP transition hooks with the live VIP state",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runVRRPSync(configPath)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", defaultConfigPath, "path to config file")
	return cmd
}

func runVRRPSync(configPath string) error {
	cfg, err := config.Load(configPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	return svc.SyncVRRPHooks(cfg)
}
