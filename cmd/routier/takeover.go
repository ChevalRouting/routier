package main

import (
	"fmt"
	"os"

	"github.com/ChevalRouting/routier/pkg/host/takeover"
	"github.com/spf13/cobra"
)

func newTakeoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "takeover",
		Short: "back up host network config and hand networking to routier",
		RunE:  runTakeover,
	}
	cmd.Flags().Bool("force", false, "take over even without a routier config")

	return cmd
}

func runTakeover(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")
	if _, err := os.Stat(defaultConfigPath); err != nil && !force {
		return fmt.Errorf("no %s yet: apply a configuration first, or pass --force", defaultConfigPath)
	}

	dir, err := takeover.Run()
	if err != nil {
		return err
	}

	_, _ = fmt.Printf("routier takeover: done, native config backed up under %s\n", dir)
	return nil
}
