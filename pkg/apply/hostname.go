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

	hostsData, _ := os.ReadFile("/etc/hosts")
	wantHosts := desiredHosts(hostname, hostsData)
	hostsOK := string(hostsData) == wantHosts

	if fileOK && kernelOK && hostsOK {
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

	if !hostsOK {
		if err := os.WriteFile("/etc/hosts", []byte(wantHosts), 0644); err != nil {
			return fmt.Errorf("write /etc/hosts: %w", err)
		}
	}

	log.Info().Str("hostname", hostname).Msg("set hostname")
	return nil
}

func desiredHosts(hostname string, current []byte) string {
	v4 := "127.0.0.1\tlocalhost localhost.localdomain " + hostname
	v6 := "::1\tlocalhost localhost.localdomain " + hostname

	var kept []string
	for _, line := range strings.Split(string(current), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 1 && (fields[0] == "127.0.0.1" || fields[0] == "127.0.1.1" || fields[0] == "::1") {
			continue
		}

		kept = append(kept, line)
	}

	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}

	return strings.Join(append([]string{v4, v6}, kept...), "\n") + "\n"
}
