package svc

import (
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/net/netlink"
)

type vpnPlan struct {
	wireguard []string
	removed   []string
}

var reconcileVPNTunnels = netlink.ReconcileTunnels

func planVPN(names []string, cfg *config.Config) vpnPlan {
	plan := vpnPlan{}
	queued := map[string]bool{}
	for _, name := range names {
		if strings.HasPrefix(name, "wireguard/") && strings.HasSuffix(name, ".conf") {
			wg := strings.TrimSuffix(strings.TrimPrefix(name, "wireguard/"), ".conf")
			queued[wg] = true
		}
	}

	if cfg != nil {
		for name := range cfg.VPN.Wireguard {
			if !netlink.LinkExists(name) {
				queued[name] = true
			}
		}

		seen := map[string]bool{}
		for _, name := range append(netlink.ManagedWireguardLinks(), managedWireguardConfs()...) {
			if cfg.VPN.Wireguard[name] != nil || seen[name] {
				continue
			}

			seen[name] = true
			plan.removed = append(plan.removed, name)
		}
	}

	for name := range queued {
		plan.wireguard = append(plan.wireguard, name)
	}

	sort.Strings(plan.wireguard)
	sort.Strings(plan.removed)
	return plan
}

func applyVPN(cfg *config.Config, plan vpnPlan, dryRun bool, warnings *[]string) error {
	if cfg != nil {
		if err := reconcileVPNTunnels(cfg, dryRun); err != nil {
			return err
		}
	}

	var actions []Action
	for _, name := range plan.removed {
		actions = append(actions, Action{
			Desc:       "wireguard down " + name + " (removed)",
			Args:       []string{"wg-quick", "down", name},
			BestEffort: true,
		})
	}

	for _, name := range plan.wireguard {
		actions = append(actions,
			Action{Desc: "wireguard down " + name, Args: []string{"wg-quick", "down", name}, BestEffort: true},
			Action{Desc: "wireguard up " + name, Args: []string{"wg-quick", "up", name}},
		)
	}

	if err := Reload(actions, dryRun, warnings); err != nil {
		return err
	}

	if !dryRun {
		for _, name := range plan.removed {
			pruneWireguardConf(name)
		}
	}

	return nil
}
