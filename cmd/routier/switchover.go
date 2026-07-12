package main

import (
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/spf13/cobra"
)

func newSwitchoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "switchover",
		Short: "run VRRP BACKUP state actions (bring down WireGuard interfaces)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return svc.WireguardDown()
		},
	}
}

func newActivateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "activate",
		Short: "run VRRP MASTER state actions (bring up WireGuard interfaces)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return svc.WireguardUp()
		},
	}
}
