package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ChevalRouting/routier/pkg/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var VERSION = "dev"

func main() {
	ensureRoot()

	log.Logger = logging.Console()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if base := filepath.Base(os.Args[0]); base == "setup-routier" {
		os.Args = append([]string{os.Args[0], "setup"}, os.Args[1:]...)
	}

	if err := newRootCommand().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func ensureRoot() {
	if os.Geteuid() != 0 || os.Getuid() == 0 {
		return
	}

	if err := syscall.Setreuid(0, 0); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "routier: failed to become root (setuid helper): %v\n", err)
	}
}
