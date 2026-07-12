package main

import (
	"github.com/ChevalRouting/routier/pkg/motd"
	"github.com/spf13/cobra"
)

func newWriteMOTDCommand() *cobra.Command {
	var user, pass string
	cmd := &cobra.Command{
		Use:    "write-motd",
		Short:  "write /etc/motd (version + interfaces + optional credentials)",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			motd.Write(user, pass)
			return nil
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "username for credentials section")
	cmd.Flags().StringVar(&pass, "pass", "", "password for credentials section")
	return cmd
}
