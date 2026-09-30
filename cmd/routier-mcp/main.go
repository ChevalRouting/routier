package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	VERSION = "dev"
	RELEASE = "0"
)

func main() {
	log.Logger = zerolog.New(consoleWriter()).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if err := serveCmd().Execute(); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}

func version() string {
	return fmt.Sprintf("%s-r%s", VERSION, RELEASE)
}

func consoleWriter() zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
		FormatErrFieldValue: func(v any) string {
			s, _ := v.(string)
			if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
				if unquoted, err := strconv.Unquote(s); err == nil {
					s = unquoted
				}
			}

			lines := strings.Split(s, "\n")
			if len(lines) <= 1 {
				return s
			}

			return "\n  " + strings.Join(lines, "\n  ")
		},
	}
}
