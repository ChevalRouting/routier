package main

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	VERSION = "dev"
	RELEASE = "0"
)

func main() {
	log.Logger = logging.Console()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if err := serveCmd().Execute(); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

func version() string {
	return fmt.Sprintf("%s-r%s", VERSION, RELEASE)
}
