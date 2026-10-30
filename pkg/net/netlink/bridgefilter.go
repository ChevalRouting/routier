//go:build linux

package netlink

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

const sysctlRoot = "/proc/sys"

var bridgeNetfilterKeys = []string{"net.bridge.bridge-nf-call-iptables", "net.bridge.bridge-nf-call-ip6tables", "net.bridge.bridge-nf-call-arptables"}

func managesBridges(cfg *config.Config) bool {
	for _, iface := range cfg.Interfaces {
		switch iface.Type {
		case "bridge", "vxlan-bridge", "vnet":
			return true
		}
	}

	return false
}

func keepBridgedFramesOutOfNetfilter(cfg *config.Config, root string, dryRun bool) {
	if !managesBridges(cfg) {
		return
	}

	for _, key := range bridgeNetfilterKeys {
		ensureSysctl(cfg, root, key, "0", "keep bridged frames out of netfilter", dryRun)
	}
}

func ensureSysctl(cfg *config.Config, root, key, value, reason string, dryRun bool) {
	if _, set := cfg.Sysctl[key]; set {
		return
	}

	path := filepath.Join(root, strings.ReplaceAll(key, ".", string(filepath.Separator)))
	current, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(current)) == value {
		return
	}

	if dryRun {
		log.Info().Str("sysctl", key).Str("value", value).Msg("would " + reason)
		return
	}

	if err := os.WriteFile(path, []byte(value+"\n"), 0o644); err != nil {
		log.Warn().Err(err).Str("sysctl", key).Msg(reason)
		return
	}

	log.Info().Str("sysctl", key).Str("value", value).Msg(reason)
}
