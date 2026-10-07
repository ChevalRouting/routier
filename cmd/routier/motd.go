package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/host/motd"
	"github.com/spf13/cobra"
)

func newWriteMOTDCommand() *cobra.Command {
	var user, pass string
	cmd := &cobra.Command{
		Use:    "write-motd",
		Short:  "write /etc/motd (version + interfaces + optional credentials)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			motd.Write(cmd.Context(), user, pass)
			return nil
		},
	}

	cmd.Flags().StringVar(&user, "user", "", "username for credentials section")
	cmd.Flags().StringVar(&pass, "pass", "", "password for credentials section")
	return cmd
}

func newIssueCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "issue",
		Short:  "print the console login banner (ascii logo + version)",
		Hidden: true,
		Run: func(cmd *cobra.Command, _ []string) {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), motd.RenderIssue(VERSION))
		},
	}
}
