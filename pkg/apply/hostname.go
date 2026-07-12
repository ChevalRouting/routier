package apply

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

func Hostname(hostname string, dryRun bool) error {
	if hostname == "" {
		return nil
	}

	fileOK := false
	if data, err := os.ReadFile("/etc/hostname"); err == nil {
		fileOK = strings.TrimRight(string(data), "\n") == hostname
	}

	kernelOK := false
	if out, err := exec.Command("hostname").Output(); err == nil {
		kernelOK = strings.TrimRight(string(out), "\n") == hostname
	}

	if fileOK && kernelOK {
		return nil
	}

	if dryRun {
		log.Info().Str("hostname", hostname).Msg("would set hostname")
		return nil
	}

	if !fileOK {
		if err := os.WriteFile("/etc/hostname", []byte(hostname+"\n"), 0644); err != nil {
			return fmt.Errorf("write /etc/hostname: %w", err)
		}
	}

	if !kernelOK {
		out, err := exec.Command("hostname", hostname).CombinedOutput()
		if err != nil {
			return fmt.Errorf("hostname %s: %w: %s", hostname, err, strings.TrimSpace(string(out)))
		}
	}

	log.Info().Str("hostname", hostname).Msg("set hostname")
	return nil
}
