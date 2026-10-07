package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ChevalRouting/routier/pkg/state/applylog"
	"github.com/ChevalRouting/routier/pkg/state/failures"
	"github.com/spf13/cobra"
)

func newFailuresCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "failures",
		Short: "inspect preserved rendered artifacts from failed applies",
	}

	cmd.AddCommand(newFailuresListCommand(), newFailuresShowCommand(), newFailuresExportCommand(), newFailuresRemoveCommand())
	return cmd
}

func newFailuresListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list preserved failed-apply bundles",
		RunE:  newFailuresListCommandHandler,
	}
}

func newFailuresShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "show a failed-apply bundle",
		Args:  cobra.ExactArgs(1),
		RunE:  newFailuresShowCommandHandler,
	}
}

func newFailuresExportCommand() *cobra.Command {
	var out string
	var noRedact bool
	cmd := &cobra.Command{
		Use:   "export <id>",
		Short: "export a failed-apply bundle as a tar.gz for sharing",
		Args:  cobra.ExactArgs(1),
		RunE: func(unusedArg2 *cobra.Command, args []string) error {
			return newFailuresExportCommandCallback(out, noRedact, unusedArg2, args)
		},
	}

	cmd.Flags().StringVarP(&out, "out", "o", "", "write the archive to this file instead of stdout")
	cmd.Flags().BoolVar(&noRedact, "no-redact", false, "do not redact secrets (keys, passwords) from the archive")
	return cmd
}

func newFailuresRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "delete a failed-apply bundle",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return failures.Remove(args[0])
		},
	}
}

func newFailuresListCommandHandler(_ *cobra.Command, _ []string) error {
	metas := failures.List()
	if len(metas) == 0 {
		_, _ = fmt.Println("no failures")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ID\tTIME\tSOURCE\tERRORS\tARTIFACTS")
	for _, m := range metas {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\n", m.ID, m.Time.Format(time.RFC3339), m.Source, len(m.Errors), len(m.Artifacts))
	}

	return tw.Flush()
}

func newFailuresShowCommandHandler(_ *cobra.Command, args []string) error {
	m, err := failures.Get(args[0])
	if err != nil {
		return err
	}

	_, _ = fmt.Printf("id:      %s\n", m.ID)
	_, _ = fmt.Printf("time:    %s\n", m.Time.Format(time.RFC3339))
	_, _ = fmt.Printf("source:  %s\n", m.Source)
	if m.SnapID != "" {
		_, _ = fmt.Printf("snap:    %s\n", m.SnapID)
	}

	_, _ = fmt.Printf("bundle:  %s\n", failures.Dir(m.ID))
	_, _ = fmt.Printf("log:     %s\n", applylog.Path(m.ID))
	_, _ = fmt.Printf("inspect: routier history show %s\n", m.ID)

	_, _ = fmt.Println("\nerrors:")
	for _, e := range m.Errors {
		loc := e.Dest
		if e.Line > 0 {
			loc = fmt.Sprintf("%s:%d", e.Dest, e.Line)
		}

		_, _ = fmt.Printf("  %s: %s\n", loc, e.Message)
	}

	_, _ = fmt.Println("\nrendered artifacts:")
	for _, a := range m.Artifacts {
		_, _ = fmt.Printf("  %s\n", a)
	}

	return nil
}

func newFailuresExportCommandCallback(out string, noRedact bool, _ *cobra.Command, args []string) error {
	w := os.Stdout
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			return err
		}

		defer func(action func() error) { _ = action() }(f.Close)
		w = f
	}

	return failures.Export(args[0], w, !noRedact)
}
