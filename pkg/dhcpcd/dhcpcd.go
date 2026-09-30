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
	dhcpcdBin = "dhcpcd"

	sysctlAcceptRA = "/proc/sys/net/ipv6/conf/%s/accept_ra"
	sysctlAutoconf = "/proc/sys/net/ipv6/conf/%s/autoconf"
	slaacAcceptRA  = "2"
	slaacAutoconf  = "1"
	offAcceptRA    = "0"
	offAutoconf    = "0"
)

func pidFile(dev, afi string) string {
	out, err := exec.Command(dhcpcdBin, "-"+afi, "-P", dev).Output()
	if err != nil {
		log.Debug().Str("dev", dev).Str("afi", afi).Err(err).Msg("dhcpcd: `-P` print pidfile failed")
		return ""
	}

	path := strings.TrimSpace(string(out))
	log.Debug().Str("dev", dev).Str("afi", afi).Str("pidfile", path).Msg("dhcpcd: resolved pidfile via `-P`")
	return path
}

func runningPid(path string) (int, bool) {
	log.Debug().Str("pidfile", path).Msg("dhcpcd: reading pidfile")
	data, err := os.ReadFile(path)
	if err != nil {
		log.Debug().Str("pidfile", path).Err(err).Msg("dhcpcd: pidfile not readable")
		return 0, false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		log.Debug().Str("pidfile", path).Str("contents", strings.TrimSpace(string(data))).Err(err).Msg("dhcpcd: pidfile has no valid pid")
		return 0, false
	}

	alive := syscall.Kill(pid, 0) == nil
	log.Debug().Str("pidfile", path).Int("pid", pid).Bool("alive", alive).Msg("dhcpcd: signal(0) liveness check")
	return pid, alive
}

func running(dev, afi string) bool {
	path := pidFile(dev, afi)
	if path == "" {
		return false
	}

	_, ok := runningPid(path)
	return ok
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

	stopUnwanted(desired, dryRun)

	for dev, w := range desired {
		syncDHCPClient(dev, "4", w.v4, dryRun)
		syncDHCPClient(dev, "6", w.v6, dryRun)

		if w.slaac {
			syncSLAAC(dev, dryRun)
		} else {
			disableSLAAC(dev, dryRun)
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
		args := []string{"-" + afi, "-b", dev}
		if err := exec.Command(dhcpcdBin, args...).Run(); err != nil {
			log.Warn().Err(err).Str("dev", dev).Str("afi", afi).Msg("start dhcpcd")
		}
	case !want && isRunning:
		if dryRun {
			log.Info().Str("dev", dev).Str("afi", afi).Msg("would stop dhcpcd")
			return
		}

		log.Info().Str("dev", dev).Str("afi", afi).Msg("stop dhcpcd")
		if err := exec.Command(dhcpcdBin, "-"+afi, "-k", dev).Run(); err != nil {
			log.Warn().Err(err).Str("dev", dev).Str("afi", afi).Msg("stop dhcpcd")
		}
	}
}

func stopUnwanted(desired map[string]wantedProto, dryRun bool) {
	wanted := map[string]bool{}
	var dir string
	for dev, w := range desired {
		if w.v4 {
			if p := pidFile(dev, "4"); p != "" {
				wanted[p] = true
				dir = filepath.Dir(p)
			}
		}

		if w.v6 {
			if p := pidFile(dev, "6"); p != "" {
				wanted[p] = true
				dir = filepath.Dir(p)
			}
		}
	}

	if dir == "" {
		dir = pidDir()
	}

	if dir == "" {
		return
	}

	entries, _ := filepath.Glob(filepath.Join(dir, "*.pid"))
	for _, path := range entries {
		if !wanted[path] {
			stopOrphan(path, dryRun)
		}
	}
}

func pidDir() string {
	out, err := exec.Command(dhcpcdBin, "-P").Output()
	if err != nil {
		return ""
	}

	return filepath.Dir(strings.TrimSpace(string(out)))
}

func stopOrphan(path string, dryRun bool) {
	pid, ok := runningPid(path)
	if !ok {
		return
	}

	if dryRun {
		log.Info().Str("pidfile", path).Msg("would stop orphan dhcpcd")
		return
	}

	log.Info().Str("pidfile", path).Int("pid", pid).Msg("stop orphan dhcpcd")
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Warn().Err(err).Int("pid", pid).Msg("dhcpcd: failed to signal orphan")
	}
}

func syncSLAAC(dev string, dryRun bool) {
	if dryRun {
		log.Info().Str("dev", dev).Msg("would enable SLAAC")
		return
	}

	writeSLAAC(dev, slaacAcceptRA, slaacAutoconf)
}

func disableSLAAC(dev string, dryRun bool) {
	if dryRun {
		log.Info().Str("dev", dev).Msg("would disable router advertisements")
		return
	}

	writeSLAAC(dev, offAcceptRA, offAutoconf)
}

var writeSysctl = func(path, val string) error {
	return os.WriteFile(path, []byte(val), 0644)
}

func writeSLAAC(dev, acceptRA, autoconf string) {
	sysctls := map[string]string{
		fmt.Sprintf(sysctlAcceptRA, dev): acceptRA + "\n",
		fmt.Sprintf(sysctlAutoconf, dev): autoconf + "\n",
	}

	for path, val := range sysctls {
		if err := writeSysctl(path, val); err != nil {
			log.Debug().Err(err).Str("dev", dev).Str("sysctl", path).Msg("set SLAAC sysctl")
		}
	}
}
