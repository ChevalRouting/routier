package svc

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/kea"
	"github.com/ChevalRouting/routier/pkg/netlink"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/vtysh"
	"github.com/rs/zerolog/log"
)

type Action struct {
	Desc       string
	Args       []string
	Fallback   []string
	Timeout    time.Duration
	BestEffort bool
}

func runCmd(args []string, timeout time.Duration) error {
	out, err := runCombined(args, timeout)
	if len(out) > 0 {
		_, _ = os.Stdout.Write(out)
	}

	return err
}

func runCmdOutput(args []string, timeout time.Duration) error {
	out, err := runCombined(args, timeout)
	if err != nil && len(out) > 0 {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}

	return err
}

func Reload(actions []Action, dryRun bool) error {
	for _, a := range actions {
		if dryRun {
			log.Info().Strs("cmd", a.Args).Msgf("would: %s", a.Desc)
			continue
		}

		log.Info().Msg(a.Desc)
		if err := runCmdOutput(a.Args, a.Timeout); err != nil {
			if a.BestEffort {
				log.Warn().Err(err).Msgf("%s failed (best-effort, continuing)", a.Desc)
				continue
			}

			if len(a.Fallback) == 0 {
				return fmt.Errorf("%s: %w", a.Desc, err)
			}

			log.Warn().Err(err).Msgf("%s failed, falling back to: %s", a.Desc, strings.Join(a.Fallback, " "))
			if err2 := runCmdOutput(a.Fallback, 0); err2 != nil {
				return fmt.Errorf("%s (fallback): %w", a.Desc, err2)
			}
		}
	}

	return nil
}

type reloadConfig struct {
	Sysctl      bool
	Nftables    bool
	FRRConf     bool
	FRRDaemons  bool
	Keepalived  bool
	RADVD       bool
	Conntrackd  bool
	SSH         bool
	ModulesLoad bool
	GAI         bool
	KeaDHCP4    bool
	KeaDHCP6    bool
	Wireguard   []string
}

func ReloadFromOutputs(names []string, cfg *config.Config, dryRun bool, skipWireguard bool) error {
	rc := reloadConfig{}

	for _, n := range names {
		switch n {
		case "sysctl/routier.conf":
			rc.Sysctl = true
		case "nftables/routier.nft":
			rc.Nftables = true
		case "frr/frr.conf":
			rc.FRRConf = true
		case "frr/daemons":
			rc.FRRDaemons = true
		case "keepalived/keepalived.conf":
			rc.Keepalived = true
		case "radvd/radvd.conf":
			rc.RADVD = true
		case "conntrackd/conntrackd.conf":
			rc.Conntrackd = true
		case "ssh/sshd_config":
			rc.SSH = true
		case "modules-load.d/routier.conf":
			rc.ModulesLoad = true
		case "gai/gai.conf":
			rc.GAI = true
		case "kea/kea-dhcp4.conf":
			rc.KeaDHCP4 = true
		case "kea/kea-dhcp6.conf":
			rc.KeaDHCP6 = true
		}

		if strings.HasPrefix(n, "wireguard/") && strings.HasSuffix(n, ".conf") {
			wg := strings.TrimSuffix(strings.TrimPrefix(n, "wireguard/"), ".conf")
			rc.Wireguard = append(rc.Wireguard, wg)
		}
	}

	if skipWireguard {
		rc.Wireguard = nil
	}

	if cfg != nil && !skipWireguard {
		queued := make(map[string]bool, len(rc.Wireguard))
		for _, wg := range rc.Wireguard {
			queued[wg] = true
		}

		for name := range cfg.Wireguard {
			if queued[name] {
				continue
			}

			if !netlink.LinkExists(name) {
				log.Info().Str("interface", name).Msg("wireguard interface missing, bringing it up")
				rc.Wireguard = append(rc.Wireguard, name)
			}
		}
	}

	var pre []Action
	if rc.Sysctl {
		pre = append(pre, Action{Desc: "apply sysctl", Args: []string{"sysctl", "-p", "/etc/sysctl.d/99-routier.conf"}})
	}

	if cfg != nil {
		if dryRun {
			log.Info().Msg("would: reconcile network")
		} else if err := netlink.Reconcile(cfg, dryRun); err != nil {
			return err
		}
	}

	if rc.Nftables {
		pre = append(pre,
			Action{Desc: "validate nftables", Args: []string{"nft", "-c", "-f", "/etc/nftables.d/routier.nft"}},
			Action{Desc: "load nftables", Args: []string{"nft", "-f", "/etc/nftables.d/routier.nft"}},
		)
	}

	if err := Reload(pre, dryRun); err != nil {
		return err
	}

	var actions []Action
	if rc.FRRConf || rc.FRRDaemons {
		if rc.FRRDaemons && frrStatus() == svcStarted {

			if dryRun {
				log.Info().Msg("would: validate frr config")
				log.Info().Msg("would: restart frr (daemons changed)")
			} else {
				if err := runCmdOutput([]string{"vtysh", "-C", "-f", "/etc/frr/frr.conf"}, 0); err != nil {
					return err
				}

				if err := frrRestart(); err != nil {
					return err
				}
			}
		} else if rc.FRRDaemons {
			if dryRun {
				log.Info().Msg("would: start frr (daemons changed)")
			} else if err := frrStart(); err != nil {
				return err
			}
		} else if frrStatus() == svcStarted {
			if script := frrReloadScript(); script != "" {
				actions = append(actions, Action{
					Desc: "validate frr config",
					Args: []string{"vtysh", "-C", "-f", "/etc/frr/frr.conf"},
				})
				actions = append(actions, Action{
					Desc:    "reload frr config",
					Args:    []string{script, "--reload", "/etc/frr/frr.conf"},
					Timeout: 60 * time.Second,
				})
			} else {
				if dryRun {
					log.Info().Msg("would: validate frr config")
					log.Info().Msg("would: restart frr (no reload script)")
				} else {
					if err := runCmdOutput([]string{"vtysh", "-C", "-f", "/etc/frr/frr.conf"}, 0); err != nil {
						return err
					}

					if err := frrRestart(); err != nil {
						return err
					}
				}
			}
		} else {
			if dryRun {
				log.Info().Msg("would: start frr")
			} else if frrStatus() == svcCrashed {
				if err := frrRestart(); err != nil {
					return err
				}
			} else {
				if err := frrStart(); err != nil {
					return err
				}
			}
		}
	}

	if rc.Keepalived {
		if ServiceRunning("keepalived") {
			actions = append(actions, Action{Desc: "reload keepalived", Args: []string{"rc-service", "keepalived", "reload"}})
		} else {
			actions = append(actions, startAction("keepalived"))
		}
	}

	if rc.RADVD {
		actions = append(actions, Action{Desc: "validate radvd config", Args: []string{"radvd", "-C", "/etc/radvd.conf", "-c"}})
		if ServiceRunning("radvd") {
			actions = append(actions, Action{Desc: "reload radvd", Args: []string{"rc-service", "radvd", "reload"}})
		} else {
			actions = append(actions, startAction("radvd"))
		}
	}

	if rc.Conntrackd {
		if ServiceRunning("conntrackd") {
			actions = append(actions, Action{Desc: "restart conntrackd", Args: []string{"rc-service", "conntrackd", "restart"}, BestEffort: true})
		} else {
			a := startAction("conntrackd")
			a.BestEffort = true
			actions = append(actions, a)
		}
	}

	if rc.SSH {
		actions = append(actions,
			Action{Desc: "validate sshd config", Args: []string{"sshd", "-t"}},
			Action{Desc: "reload sshd", Args: []string{"rc-service", "sshd", "reload"}},
		)
	}

	if rc.KeaDHCP4 || rc.KeaDHCP6 {
		actions = append(actions,
			Action{Desc: "ensure kea log dir", Args: []string{"install", "-d", "-m", "0750", "-o", "kea", "-g", "kea", "/var/log/kea"}, BestEffort: true},
			Action{Desc: "ensure kea run dir", Args: []string{"install", "-d", "-m", "0750", "-o", "kea", "-g", "kea", "/run/kea"}, BestEffort: true},
		)
	}

	if rc.ModulesLoad && cfg != nil {
		for _, mod := range cfg.BootModules {
			actions = append(actions, Action{
				Desc:       "load kernel module " + mod,
				Args:       []string{"modprobe", mod},
				BestEffort: true,
			})
		}
	}

	if cfg != nil {
		desired := make(map[string]bool, len(cfg.Wireguard))
		for name := range cfg.Wireguard {
			desired[name] = true
		}

		for _, wg := range netlink.ManagedWireguardLinks() {
			if desired[wg] {
				continue
			}

			actions = append(actions, Action{
				Desc:       "wireguard down " + wg + " (removed)",
				Args:       []string{"wg-quick", "down", wg},
				BestEffort: true,
			})
		}
	}

	for _, wg := range rc.Wireguard {
		actions = append(actions,
			Action{
				Desc:       "wireguard down " + wg,
				Args:       []string{"wg-quick", "down", wg},
				BestEffort: true,
			},
			Action{
				Desc: "wireguard up " + wg,
				Args: []string{"wg-quick", "up", wg},
			},
		)
	}

	if err := Reload(actions, dryRun); err != nil {
		return err
	}

	if !dryRun {
		if rc.KeaDHCP4 {
			if err := applyKeaServer("kea-dhcp4", "dhcp4"); err != nil {
				return err
			}
		}

		if rc.KeaDHCP6 {
			if err := applyKeaServer("kea-dhcp6", "dhcp6"); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyKeaServer(svcName, service string) error {
	file := "/etc/kea/" + svcName + ".conf"
	if err := runCmdOutput([]string{svcName, "-t", file}, 0); err != nil {
		return fmt.Errorf("kea %s: config invalid: %w", svcName, err)
	}

	if !ServiceRunning(svcName) {
		if err := runCmd([]string{"rc-service", svcName, "start"}, 0); err != nil {
			return fmt.Errorf("kea %s: start failed: %w", svcName, err)
		}

		return nil
	}

	if client, err := kea.LoadLocal(); err == nil {
		if err := client.ConfigReload(service); err == nil {
			log.Info().Str("service", svcName).Msg("kea: config reloaded")
			return nil
		} else {
			log.Warn().Err(err).Str("service", svcName).Msg("kea: config-reload failed, restarting")
		}
	}

	if err := runCmd([]string{"rc-service", svcName, "restart"}, 0); err != nil {
		return fmt.Errorf("kea %s: restart failed: %w", svcName, err)
	}

	return nil
}

type svcStatus int

const (
	svcStopped svcStatus = iota
	svcStarted
	svcCrashed
)

func getServiceStatus(name string) svcStatus {
	out, _ := runCombined([]string{"rc-service", "--nocolor", name, "status"}, 0)

	combined := strings.ToLower(string(out))
	if strings.Contains(combined, "status: started") || strings.Contains(combined, "status: starting") {
		return svcStarted
	}

	if strings.Contains(combined, "status: crashed") {
		return svcCrashed
	}

	return svcStopped
}

func ServiceRunning(name string) bool {
	return getServiceStatus(name) == svcStarted
}

func frrStatus() svcStatus {
	outb, _ := runCombined([]string{"rc-service", "--nocolor", "frr", "status"}, 0)

	output := string(outb)
	lower := strings.ToLower(output)

	if strings.Contains(lower, "status: starting") {
		return svcStarted
	}

	hasRunning := false
	hasFailed := false
	for line := range strings.SplitSeq(output, "\n") {
		lineLower := strings.ToLower(strings.TrimSpace(line))
		if !strings.HasPrefix(lineLower, "status of ") {
			continue
		}

		if strings.HasSuffix(lineLower, ": running") {
			hasRunning = true
		} else if strings.HasSuffix(lineLower, ": failed") {
			hasFailed = true
		}
	}

	if hasRunning {
		return svcStarted
	}

	if hasFailed {
		return svcCrashed
	}

	return svcStopped
}

func startAction(name string) Action {
	if getServiceStatus(name) == svcCrashed {
		return Action{Desc: "restart " + name + " (crashed)", Args: []string{"rc-service", name, "restart"}}
	}

	return Action{Desc: "start " + name, Args: []string{"rc-service", name, "start"}}
}

func frrWaitReady(timeout time.Duration) error {
	return vtysh.WaitReady(timeout)
}

func frrStart() error {
	log.Info().Msg("start frr")
	if err := runCmd([]string{"rc-service", "frr", "start"}, 90*time.Second); err == nil {
		return nil
	}

	log.Warn().Msg("frr start script did not complete cleanly, checking if daemons are up")
	return frrWaitReady(30 * time.Second)
}

func frrRestart() error {
	log.Info().Msg("restart frr")
	if err := runCmd([]string{"rc-service", "frr", "restart"}, 90*time.Second); err == nil {
		return nil
	}

	log.Warn().Msg("frr restart script did not complete cleanly, checking if daemons are up")
	return frrWaitReady(30 * time.Second)
}

func frrReloadScript() string {
	for _, candidate := range []string{"/usr/lib/frr/frr-reload.py", "/usr/lib/frr/frr-reload"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

type serviceEntry struct {
	name       string
	needed     bool
	bestEffort bool
}

func ReconcileServices(cfg *config.Config, dryRun bool) error {
	needFRR := cfg.Routing != nil

	needKeepalived := false
	for _, iface := range cfg.Interfaces {
		if len(iface.VRRP) > 0 {
			needKeepalived = true
			break
		}
	}

	needRAVD := cfg.Routing != nil &&
		cfg.Routing.RADVD != nil &&
		len(cfg.Routing.RADVD.Interfaces) > 0

	needConntrackd := cfg.Conntrackd != nil

	needKea := render.LocalKeaEnabled(cfg)
	needKeaV4 := needKea && len(cfg.DHCP.Subnets4) > 0
	needKeaV6 := needKea && len(cfg.DHCP.Subnets6) > 0

	frrSt := frrStatus()
	switch {
	case needFRR && frrSt != svcStarted:
		if dryRun {
			log.Info().Msgf("would: start frr (state: %v)", frrSt)
		} else {
			log.Info().Msgf("start frr (state: %v)", frrSt)
			if frrSt == svcCrashed {
				if err := frrRestart(); err != nil {
					return err
				}
			} else {
				if err := frrStart(); err != nil {
					return err
				}
			}
		}
	case !needFRR && frrSt == svcStarted:
		if dryRun {
			log.Info().Msg("would: stop frr (no longer needed)")
		} else {
			log.Info().Msg("stop frr (no longer needed)")
			if err := runCmd([]string{"rc-service", "frr", "stop"}, 0); err != nil {
				log.Warn().Err(err).Msg("stop frr failed")
			}
		}
	}

	services := []serviceEntry{
		{name: "routier-ui", needed: true},
		{name: "keepalived", needed: needKeepalived},
		{name: "radvd", needed: needRAVD},
		{name: "conntrackd", needed: needConntrackd, bestEffort: true},
		{name: "kea-dhcp4", needed: needKeaV4, bestEffort: true},
		{name: "kea-dhcp6", needed: needKeaV6, bestEffort: true},
	}

	for _, s := range services {
		status := getServiceStatus(s.name)
		switch {
		case s.needed && status != svcStarted:
			verb := "start"
			if status == svcCrashed {
				verb = "restart"
			}

			if dryRun {
				log.Info().Msgf("would: %s %s (needed, state: %s)", verb, s.name, map[svcStatus]string{svcStopped: "stopped", svcCrashed: "crashed"}[status])
				continue
			}

			log.Info().Msgf("%s %s (state: %s)", verb, s.name, map[svcStatus]string{svcStopped: "stopped", svcCrashed: "crashed"}[status])
			if err := runCmd([]string{"rc-service", s.name, verb}, 0); err != nil {
				if s.bestEffort {
					log.Warn().Err(err).Msgf("%s %s failed (best-effort, continuing)", verb, s.name)
					continue
				}

				return fmt.Errorf("%s %s: %w", verb, s.name, err)
			}
		case !s.needed && status == svcStarted:
			if dryRun {
				log.Info().Msgf("would: stop %s (no longer needed)", s.name)
				continue
			}

			log.Info().Msgf("stop %s (no longer needed)", s.name)
			if err := runCmd([]string{"rc-service", s.name, "stop"}, 0); err != nil {
				log.Warn().Err(err).Msgf("stop %s failed", s.name)
			}
		}

		setBootRunlevel(s.name, s.needed, dryRun)
	}

	setBootRunlevel("frr", needFRR, dryRun)
	return nil
}

func setBootRunlevel(name string, enabled, dryRun bool) {
	if dryRun {
		return
	}

	if inDefaultRunlevel(name) == enabled {
		return
	}

	verb := "add"
	if !enabled {
		verb = "del"
	}

	if err := runCmd([]string{"rc-update", verb, name, "default"}, 0); err != nil {
		log.Warn().Err(err).Msgf("rc-update %s %s failed", verb, name)
	}
}

func inDefaultRunlevel(name string) bool {
	out, err := runCombined([]string{"rc-update", "show", "default"}, 0)
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(out), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == name {
			return true
		}
	}

	return false
}
