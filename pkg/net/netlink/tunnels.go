//go:build linux

package netlink

import (
	"fmt"
	"sort"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	vnl "github.com/vishvananda/netlink"
)

func ReconcileTunnels(cfg *config.Config, dryRun bool) error {
	all, err := vnl.LinkList()
	if err != nil {
		return fmt.Errorf("list tunnel links: %w", err)
	}

	tunnels := cfg.Tunnels()
	actual := make(map[string]vnl.Link, len(all))
	for _, link := range all {
		name := link.Attrs().Name
		actual[name] = link
		if link.Attrs().Alias != managedAlias || linkKindOf(link) != kindTunnel || tunnels[name] != nil {
			continue
		}

		if dryRun {
			log.Info().Str("link", name).Msg("would delete stale tunnel")
			continue
		}

		if err := linkDel(link); err != nil {
			return fmt.Errorf("delete stale tunnel %s: %w", name, err)
		}

		delete(actual, name)
	}

	names := make([]string, 0, len(tunnels))
	for name := range tunnels {
		names = append(names, name)
	}

	sort.Strings(names)
	for _, name := range names {
		tunnel := tunnels[name]
		spec := linkSpec{kind: kindTunnel, device: name, mtu: tunnel.MTU, tunnel: tunnel}
		if err := ensureTunnel(spec, actual, dryRun); err != nil {
			return fmt.Errorf("tunnel %s: %w", name, err)
		}
	}

	return nil
}
