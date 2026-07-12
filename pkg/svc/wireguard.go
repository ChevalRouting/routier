package svc

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

func wireguardInterfaces() []string {
	entries, err := filepath.Glob("/etc/wireguard/*.conf")
	if err != nil || len(entries) == 0 {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, confPath := range entries {
		names = append(names, strings.TrimSuffix(filepath.Base(confPath), ".conf"))
	}

	return names
}

func wgQuickAll(action string) {
	ifaces := wireguardInterfaces()
	if len(ifaces) == 0 {
		log.Info().Str("action", action).Msg("wireguard: no configs found")
		return
	}

	var wg sync.WaitGroup
	for _, name := range ifaces {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			log.Info().Str("interface", name).Str("action", action).Msg("wireguard: wg-quick")
			out, err := exec.Command("wg-quick", action, name).CombinedOutput()
			if err != nil {
				log.Warn().Err(err).Str("interface", name).Str("action", action).
					Str("output", strings.TrimSpace(string(out))).
					Msg("wireguard: wg-quick returned non-zero (may already be in target state)")
			}
		}(name)
	}

	wg.Wait()
}

func WireguardUp() error {
	wgQuickAll("up")
	return nil
}

func WireguardDown() error {
	wgQuickAll("down")
	return nil
}
