//go:build linux

package netlink

import (
	"net"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	vrfUnreachableMetric = 4278198272
	localTable           = unix.RT_TABLE_LOCAL
	localRuleAfterL3mdev = 32765
)

var unreachableDefaults = []string{"0.0.0.0/0", "::/0"}

func isolateVRFs(cfg *config.Config, dryRun bool) {
	tables := map[int]bool{}
	for _, vrf := range cfg.VRFs {
		tables[vrf.Table] = true
	}

	reopenRemovedTables(tables, dryRun)
	if len(cfg.VRFs) == 0 {
		return
	}

	for name, vrf := range cfg.VRFs {
		closeVRFTable(name, vrf.Table, dryRun)
	}

	for _, family := range []int{vnl.FAMILY_V4, vnl.FAMILY_V6} {
		if err := moveLocalRule(family, dryRun); err != nil {
			log.Warn().Err(err).Int("family", family).Msg("move the local rule after the vrf rule")
		}
	}
}

func closeVRFTable(name string, table int, dryRun bool) {
	for _, dst := range unreachableDefaults {
		_, prefix, err := net.ParseCIDR(dst)
		if err != nil {
			continue
		}

		if dryRun {
			log.Info().Str("vrf", name).Str("route", "unreachable "+dst).Msg("would close the vrf table")
			continue
		}

		route := &vnl.Route{Dst: prefix, Table: table, Type: unix.RTN_UNREACHABLE, Priority: vrfUnreachableMetric}
		if err := routeReplace(route); err != nil {
			log.Warn().Err(err).Str("vrf", name).Str("route", "unreachable "+dst).Msg("close the vrf table")
		}
	}
}

func moveLocalRule(family int, dryRun bool) error {
	rules, err := vnl.RuleList(family)
	if err != nil {
		return err
	}

	moved, first := false, []vnl.Rule{}
	for _, rule := range rules {
		if rule.Table != localTable {
			continue
		}

		switch rule.Priority {
		case localRuleAfterL3mdev:
			moved = true
		case 0:
			first = append(first, rule)
		}
	}

	if dryRun {
		if len(first) > 0 {
			log.Info().Int("family", family).Msg("would move the local rule after the vrf rule")
		}

		return nil
	}

	if !moved {
		rule := vnl.NewRule()
		rule.Family, rule.Table, rule.Priority = family, localTable, localRuleAfterL3mdev
		if err := vnl.RuleAdd(rule); err != nil {
			return err
		}
	}

	for index := range first {
		if err := vnl.RuleDel(&first[index]); err != nil {
			return err
		}
	}

	return nil
}

func bindServicesInVRFs(cfg *config.Config, root string, dryRun bool) {
	if cfg.DHCP == nil || !cfg.DHCP.Enabled {
		return
	}

	for _, name := range cfg.DHCP.Interfaces {
		if iface := cfg.Interfaces[name]; iface != nil && iface.VRF != "" {
			ensureSysctl(cfg, root, "net.ipv4.ip_nonlocal_bind", "1", "let kea bind addresses in vrfs", dryRun)
			ensureSysctl(cfg, root, "net.ipv6.ip_nonlocal_bind", "1", "let kea bind addresses in vrfs", dryRun)
			return
		}
	}
}

func reopenRemovedTables(tables map[int]bool, dryRun bool) {
	for _, family := range []int{vnl.FAMILY_V4, vnl.FAMILY_V6} {
		routes, err := vnl.RouteListFiltered(family, &vnl.Route{Table: unix.RT_TABLE_UNSPEC}, vnl.RT_FILTER_TABLE)
		if err != nil {
			log.Warn().Err(err).Msg("list vrf tables")
			continue
		}

		for index := range routes {
			route := &routes[index]
			if route.Type != unix.RTN_UNREACHABLE || route.Priority != vrfUnreachableMetric || tables[route.Table] {
				continue
			}

			if dryRun {
				log.Info().Int("table", route.Table).Msg("would remove the unreachable default of a removed vrf")
				continue
			}

			if err := vnl.RouteDel(route); err != nil {
				log.Warn().Err(err).Int("table", route.Table).Msg("remove the unreachable default of a removed vrf")
			}
		}
	}
}
