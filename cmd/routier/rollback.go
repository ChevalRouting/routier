package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/apply"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newConfirmCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "confirm",
		Short: "confirm pending apply, disarm rollback",
		RunE:  newConfirmCommandHandler,
	}
}

func newRollbackCommand() *cobra.Command {
	var id string
	var offset int
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "restore previous snapshot",
		RunE: func(unusedArg2 *cobra.Command, unusedArg3 []string) error {
			return newRollbackCommandCallback(&id, offset, unusedArg2, unusedArg3)
		},
	}

	cmd.Flags().StringVarP(&id, "snapshot", "s", "", "snapshot id (default: latest)")
	cmd.Flags().IntVarP(&offset, "offset", "n", 0, "offset from latest (1=latest, 2=second latest, ...)")
	return cmd
}

func newConfirmCommandHandler(_ *cobra.Command, _ []string) error {
	id, err := managers.ConfirmPending()
	if err != nil {
		return err
	}

	log.Info().Str("snapshot", id).Msg("confirmed")

	return nil
}

func newRollbackCommandCallback(id *string, offset int, _ *cobra.Command, _ []string) error {
	if (*id) != "" && offset != 0 {
		return fmt.Errorf("--snapshot and --offset are mutually exclusive")
	}

	if offset != 0 {
		snapshots, err := apply.ListSnapshots()
		if err != nil {
			return err
		}

		idx := len(snapshots) - offset
		if idx < 0 || idx >= len(snapshots) {
			return fmt.Errorf("offset %d out of range (have %d snapshots)", offset, len(snapshots))
		}

		(*id) = snapshots[idx]
	}

	resolved, err := managers.Rollback((*id))
	if err != nil {
		return err
	}

	log.Info().Str("snapshot", resolved).Msg("rolled back")

	return nil
}
