package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ChevalRouting/routier/pkg/apply"
	"github.com/spf13/cobra"
)

func newSnapshotsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "inspect config snapshots (restore with: routier rollback -s <id>)",
	}

	cmd.AddCommand(newSnapshotsListCommand())
	return cmd
}

func newSnapshotsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list available snapshots",
		RunE: func(_ *cobra.Command, _ []string) error {
			infos, err := apply.ListSnapshotInfos()
			if err != nil {
				return err
			}

			if len(infos) == 0 {
				fmt.Println("no snapshots")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tTIME\tFILES")
			for _, s := range infos {
				fmt.Fprintf(tw, "%s\t%s\t%d\n", s.ID, s.Time.Format(time.RFC3339), len(s.Files))
			}

			return tw.Flush()
		},
	}
}
