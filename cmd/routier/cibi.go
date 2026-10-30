package main

import (
	"fmt"
	"path/filepath"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/cibi"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/spf13/cobra"
)

func newCibiCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cibi",
		Short: "join a cibi fleet manager",
	}

	cmd.AddCommand(newCibiJoinCommand())
	return cmd
}

func newCibiJoinCommand() *cobra.Command {
	var (
		join   cibi.Join
		dbPath string
	)
	cmd := &cobra.Command{
		Use:   "join <cibi-url> <code>",
		Short: "register this routier in cibi with a one-time enrollment code",
		Long: "Register this routier in cibi with a one-time enrollment code from cibi's Routiers page. " +
			"It creates an API key named cibi, sends it with this routier's identity, and cibi connects back to check both.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			join.Server, join.Code = args[0], args[1]
			return runCibiJoin(cmd, join, dbPath)
		},
	}

	cmd.Flags().StringVar(&join.URL, "url", "", "address cibi reaches this routier at (default: the address cibi sees, port --port)")
	cmd.Flags().IntVar(&join.Port, "port", 0, "port of the routier web server when --url is not given (default 8080)")
	cmd.Flags().BoolVar(&join.Insecure, "insecure", false, "accept any TLS certificate from cibi")
	cmd.Flags().StringVar(&dbPath, "db", cloudDBPath, "path to SQLite database")
	return cmd
}

func runCibiJoin(cmd *cobra.Command, join cibi.Join, dbPath string) error {
	ctx := cmd.Context()
	db, err := webdb.InitDB(ctx, dbPath)
	if err != nil {
		return err
	}

	defer func(action func() error) { _ = action() }(db.Close)
	self, err := identity.LoadOrCreate(filepath.Join(filepath.Dir(dbPath), identity.KeyFilename))
	if err != nil {
		return fmt.Errorf("load the routier identity: %w", err)
	}

	result, err := join.Run(ctx, db, self)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
	return nil
}
