package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/server/mcpserver"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	var (
		configPath string
		logLevel   string
	)

	cmd := &cobra.Command{
		Use:           "routier-mcp",
		Short:         "Routier MCP server",
		Version:       version(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, unusedArg3 []string) error {
			return serveCmdCallback(configPath, logLevel, cmd, unusedArg3)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "routier-mcp.yml", "path to the Routier MCP instance registry")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	return cmd
}

func serveCmdCallback(configPath string, logLevel string, cmd *cobra.Command, _ []string) error {
	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("invalid log level %q", logLevel)
	}

	zerolog.SetGlobalLevel(lvl)

	server, err := mcpserver.New(configPath, version())
	if err != nil {
		return err
	}

	return server.Run(cmd.Context())
}
