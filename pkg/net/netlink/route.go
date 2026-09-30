//go:build linux

package netlink

import (
	"fmt"
	"net"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	vnl "github.com/vishvananda/netlink"
)

const rtprotRoutier = vnl.RouteProtocol(100)

func resolveRouteDev(cfg *config.Config, dev string) (int, error) {
	name := dev
	if iface, ok := cfg.Interfaces[dev]; ok {
		name = iface.Device
	}

	link, err := vnl.LinkByName(name)
	if err != nil {
		return 0, fmt.Errorf("route dev %q: %w", name, err)
	}

	return link.Attrs().Index, nil
}

func buildStaticRoutes(cfg *config.Config, statics []config.StaticRoute, table int) ([]vnl.Route, error) {
	var routes []vnl.Route
	for _, sr := range statics {
		_, dst, err := net.ParseCIDR(sr.Destination)
		if err != nil {
			return nil, fmt.Errorf("static route %q: %w", sr.Destination, err)
		}

		r := vnl.Route{
			Dst:      dst,
			Protocol: rtprotRoutier,
			Priority: sr.Metric,
		}

		if table > 0 {
			r.Table = table
		}

		if sr.Via != "" {
			r.Gw = net.ParseIP(sr.Via)
			if r.Gw == nil {
				return nil, fmt.Errorf("static route %q: invalid via %q", sr.Destination, sr.Via)
			}
		}

		if sr.Dev != "" {
			idx, err := resolveRouteDev(cfg, sr.Dev)
			if err != nil {
				return nil, err
			}

			r.LinkIndex = idx
		}

		routes = append(routes, r)
	}

	return routes, nil
}

func buildDesiredRoutes(cfg *config.Config) ([]vnl.Route, error) {
	if cfg.Routing == nil {
		return nil, nil
	}

	base, err := buildStaticRoutes(cfg, cfg.Routing.Static, 0)
	if err != nil {
		return nil, err
	}

	routes := base

	for vrfName, vr := range cfg.Routing.VRFs {
		if len(vr.Static) == 0 {
			continue
		}

		vrfCfg, ok := cfg.VRFs[vrfName]
		if !ok || vrfCfg.Table == 0 {
			return nil, fmt.Errorf("VRF %q has static routes but no table declared in vrfs section", vrfName)
		}

		r, err := buildStaticRoutes(cfg, vr.Static, vrfCfg.Table)
		if err != nil {
			return nil, fmt.Errorf("vrf %s: %w", vrfName, err)
		}

		routes = append(routes, r...)
	}

	return routes, nil
}

func listRoutierRoutes() ([]vnl.Route, error) {
	filter := &vnl.Route{Protocol: rtprotRoutier}
	var all []vnl.Route

	for _, family := range []int{vnl.FAMILY_V4, vnl.FAMILY_V6} {
		routes, err := vnl.RouteListFiltered(family, filter, vnl.RT_FILTER_PROTOCOL)
		if err != nil {
			return nil, fmt.Errorf("list routes family %d: %w", family, err)
		}

		all = append(all, routes...)
	}

	return all, nil
}

func routeKey(r vnl.Route) string {
	dst := "default"
	if r.Dst != nil {
		dst = r.Dst.String()
	}

	key := dst
	if r.Table > 0 && r.Table != 254 {
		key = fmt.Sprintf("table%d:%s", r.Table, dst)
	}

	if len(r.Gw) > 0 {
		return key + " via " + r.Gw.String()
	}

	return key
}

func gwEqual(a, b net.IP) bool {
	if len(a) == 0 {
		a = nil
	}

	if len(b) == 0 {
		b = nil
	}

	if a == nil {
		return b == nil
	}

	return a.Equal(b)
}

func routesEqual(a, b vnl.Route) bool {
	if b.LinkIndex != 0 && a.LinkIndex != b.LinkIndex {
		return false
	}

	if b.Priority != 0 && a.Priority != b.Priority {
		return false
	}

	return gwEqual(a.Gw, b.Gw)
}

func reconcileRoutes(cfg *config.Config, dryRun bool) error {
	desired, err := buildDesiredRoutes(cfg)
	if err != nil {
		return err
	}

	current, err := listRoutierRoutes()
	if err != nil {
		return err
	}

	desiredMap := make(map[string]vnl.Route, len(desired))
	for _, r := range desired {
		desiredMap[routeKey(r)] = r
	}

	currentMap := make(map[string]vnl.Route, len(current))
	for _, r := range current {
		currentMap[routeKey(r)] = r
	}

	for k, r := range currentMap {
		if _, ok := desiredMap[k]; ok {
			continue
		}

		if dryRun {
			log.Info().Msgf("would: delete route %s", k)
			continue
		}

		log.Info().Msgf("delete route %s", k)
		if err := routeDel(&r); err != nil {
			log.Warn().Err(err).Msgf("delete route %s failed", k)
		}
	}

	for k, r := range desiredMap {
		cur, exists := currentMap[k]
		if !exists {
			if dryRun {
				log.Info().Msgf("would: add route %s", k)
				continue
			}

			log.Info().Msgf("add route %s", k)
			if err := routeAdd(&r); err != nil {
				return fmt.Errorf("add route %s: %w", k, err)
			}
		} else if !routesEqual(cur, r) {
			if dryRun {
				log.Info().Msgf("would: replace route %s", k)
				continue
			}

			log.Info().Msgf("replace route %s", k)
			if err := routeReplace(&r); err != nil {
				return fmt.Errorf("replace route %s: %w", k, err)
			}
		}
	}

	return nil
}
