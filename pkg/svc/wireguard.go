package svc

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

var wireguardHeader = []byte("# routier:")

func managedWireguardConfs() []string {
	entries, err := filepath.Glob("/etc/wireguard/*.conf")
	if err != nil {
		return nil
	}

	var names []string
	for _, confPath := range entries {
		data, err := os.ReadFile(confPath)
		if err != nil || !bytes.HasPrefix(data, wireguardHeader) {
			continue
		}

		names = append(names, strings.TrimSuffix(filepath.Base(confPath), ".conf"))
	}

	return names
}

func pruneWireguardConf(name string) {
	confPath := filepath.Join("/etc/wireguard", name+".conf")
	data, err := os.ReadFile(confPath)
	if err != nil || !bytes.HasPrefix(data, wireguardHeader) {
		return
	}

	if err := os.Remove(confPath); err != nil {
		log.Warn().Err(err).Str("interface", name).Msg("wireguard: remove stale conf")
		return
	}

	log.Info().Str("interface", name).Msg("wireguard: removed stale conf")
}
