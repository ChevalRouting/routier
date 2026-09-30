package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var VERSION = "dev"

func main() {
	ensureRoot()

	log.Logger = zerolog.New(consoleWriter()).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if base := filepath.Base(os.Args[0]); base == "setup-routier" {
		os.Args = append([]string{os.Args[0], "setup"}, os.Args[1:]...)
	}

	if err := newRootCommand().Execute(); err != nil {
		switch e := err.(type) {
		default:
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
}

func ensureRoot() {
	if os.Geteuid() != 0 || os.Getuid() == 0 {
		return
	}

	if err := syscall.Setreuid(0, 0); err != nil {
		fmt.Fprintf(os.Stderr, "routier: failed to become root (setuid helper): %v\n", err)
	}
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
