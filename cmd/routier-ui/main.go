package main

import (
	"runtime/debug"

	"github.com/ChevalRouting/routier/pkg/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	debug.SetGCPercent(10)
	debug.SetMemoryLimit(128 << 20)

	log.Logger = logging.Console()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cmd := serveCmd()
	if err := cmd.Execute(); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
