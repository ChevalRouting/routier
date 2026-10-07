package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ChevalRouting/routier/pkg/state/applylog"
	"github.com/spf13/cobra"
)

func newHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "inspect persisted apply logs",
	}

	cmd.AddCommand(newHistoryListCommand(), newHistoryShowCommand())
	return cmd
}

func newHistoryListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list recent apply runs",
		RunE:  newHistoryListCommandHandler,
	}
}

func newHistoryShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "print the full log for an apply run",
		Args:  cobra.ExactArgs(1),
		RunE:  newHistoryShowCommandHandler,
	}
}

func newHistoryListCommandHandler(_ *cobra.Command, _ []string) error {
	recs := applylog.List()
	if len(recs) == 0 {
		_, _ = fmt.Println("no apply logs")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ID\tSTARTED\tSOURCE\tRESULT\tWATCHDOG")
	for _, r := range recs {
		watchdog := "-"
		switch {
		case r.ConfirmedAt != nil:
			watchdog = "confirmed"
		case r.RolledBackAt != nil:
			watchdog = "rolled back"
		}

		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			r.ID, r.StartedAt.Format(time.RFC3339), r.Source, r.Result, watchdog)
	}

	return tw.Flush()
}

func newHistoryShowCommandHandler(_ *cobra.Command, args []string) error {
	text, err := applylog.Read(args[0])
	if err != nil {
		return err
	}

	_, _ = fmt.Print(text)
	return nil
}
