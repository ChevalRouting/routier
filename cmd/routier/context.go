package main

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type contextKey string

var contextKeyValue contextKey = "context"

type Context struct {
	Logger zerolog.Logger
}

func injectContext(cmd *cobra.Command, _ []string) error {
	lvl, err := zerolog.ParseLevel(cli.LogLevel)
	if err != nil {
		return err
	}

	zerolog.SetGlobalLevel(lvl)

	logger := log.Logger.Level(lvl)
	ctx := context.WithValue(cmd.Context(), contextKeyValue, &Context{Logger: logger})
	cmd.SetContext(ctx)
	return nil
}

func contextFromContext(cmd *cobra.Command) (*Context, error) {
	c, ok := cmd.Context().Value(contextKeyValue).(*Context)
	if !ok || c == nil {
		return &Context{Logger: log.Logger}, nil
	}

	return c, nil
}
