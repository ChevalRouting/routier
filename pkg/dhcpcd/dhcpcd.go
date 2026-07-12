//go:build linux

package dhcpcd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	dhcpcdBin  = "dhcpcd"
	pidFileDir = "/run"

	sysctlAcceptRA = "/proc/sys/net/ipv6/conf/%s/accept_ra"
	sysctlAutoconf = "/proc/sys/net/ipv6/conf/%s/autoconf"
	slaacAcceptRA  = "2"
	slaacAutoconf  = "1"
)

func pidFile(dev, afi string) string {
	return fmt.Sprintf("%s/dhcpcd-%s-%s.pid", pidFileDir, afi, dev)
}

func running(dev, afi string) bool {
	data, err := os.ReadFile(pidFile(dev, afi))
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}

	return syscall.Kill(pid, 0) == nil
}

type wantedProto struct {
	v4    bool
	v6    bool
	slaac bool
}

func Reconcile(cfg *config.Config, dryRun bool) error {
	desired := make(map[string]wantedProto, len(cfg.Interfaces))
	for _, iface := range cfg.Interfaces {
		var w wantedProto
		for _, addr := range iface.Addresses {
			switch addr {
			case "dhcp", "dhcp4":
				w.v4 = true
			case "dhcp6":
				w.v6 = true
			case "slaac":
				w.slaac = true
			}
		}

		desired[iface.Device] = w
	}

	entries, _ := filepath.Glob(filepath.Join(pidFileDir, "dhcpcd-*.pid"))
	for _, path := range entries {
		base := strings.TrimSuffix(filepath.Base(path), ".pid")
		parts := strings.SplitN(base, "-", 3)
		if len(parts) != 3 || parts[0] != "dhcpcd" {
			continue
		}

		afi, dev := parts[1], parts[2]
		if w, ok := desired[dev]; ok {
			if (afi == "4" && !w.v4) || (afi == "6" && !w.v6) {
				syncDHCPClient(dev, afi, false, dryRun)
			}
		} else {
			syncDHCPClient(dev, afi, false, dryRun)
		}
	}

	for dev, w := range desired {
		syncDHCPClient(dev, "4", w.v4, dryRun)
		syncDHCPClient(dev, "6", w.v6, dryRun)

		if w.slaac {
			syncSLAAC(dev, dryRun)
		}
	}

	return nil
}

func syncDHCPClient(dev, afi string, want bool, dryRun bool) {
	isRunning := running(dev, afi)

	switch {
	case want && !isRunning:
		if dryRun {
			log.Info().Str("dev", dev).Str("afi", afi).Msg("would start dhcpcd")
			return
		}

		log.Info().Str("dev", dev).Str("afi", afi).Msg("start dhcpcd")
		args := []string{"-" + afi, "-b", "-P", pidFile(dev, afi), dev}
		if err := exec.Command(dhcpcdBin, args...).Run(); err != nil {
			log.Warn().Err(err).Str("dev", dev).Str("afi", afi).Msg("start dhcpcd")
		}
	case !want && isRunning:
		if dryRun {
			log.Info().Str("dev", dev).Str("afi", afi).Msg("would stop dhcpcd")
			return
		}

		log.Info().Str("dev", dev).Str("afi", afi).Msg("stop dhcpcd")
		if err := exec.Command(dhcpcdBin, "-"+afi, "-k", "-P", pidFile(dev, afi), dev).Run(); err != nil {
			log.Warn().Err(err).Str("dev", dev).Str("afi", afi).Msg("stop dhcpcd")
		}
	}
}

func syncSLAAC(dev string, dryRun bool) {
	if dryRun {
		log.Info().Str("dev", dev).Msg("would enable SLAAC")
		return
	}

	sysctls := map[string]string{
		fmt.Sprintf(sysctlAcceptRA, dev): slaacAcceptRA + "\n",
		fmt.Sprintf(sysctlAutoconf, dev): slaacAutoconf + "\n",
	}

	for path, val := range sysctls {
		if err := os.WriteFile(path, []byte(val), 0644); err != nil {
			log.Warn().Err(err).Str("dev", dev).Str("sysctl", path).Msg("set SLAAC")
		}
	}
}
