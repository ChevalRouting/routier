//go:build linux

package netlink

import (
	"fmt"
	"net"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const rtprotKeepalived = 18

func reconcileAddrs(cfg *config.Config, dryRun bool) error {
	wantByDev := map[string][]string{}
	addWant := func(dev string, addrs []string) { addWantedAddrs(wantByDev, dev, addrs) }

	noDAD := map[string]bool{}
	for name, iface := range cfg.Interfaces {
		addWant(iface.Device, iface.Addresses)
		if iface.Type == "vnet" && iface.VNet != nil && iface.VNet.Gateway != nil {
			gateway := config.VNetLinkName(name, "g")
			addWant(gateway, gatewayAddrs(iface.VNet.Gateway))
			noDAD[gateway] = true
		}
	}

	for name, wg := range cfg.VPN.Wireguard {
		addWant(name, wg.Addresses)
	}

	for dev, cidrs := range wantByDev {
		if err := syncAddrs(dev, cidrs, noDAD[dev], dryRun); err != nil {
			log.Error().Err(err).Str("dev", dev).Msg("addr reconcile")
		}
	}

	return nil
}

var dhcpTokens = map[string]bool{
	"dhcp": true, "dhcp4": true, "dhcp6": true, "slaac": true,
}

func syncAddrs(dev string, wantCIDRs []string, noDAD, dryRun bool) error {
	link, err := vnl.LinkByName(dev)
	if err != nil && dryRun {
		return nil
	}

	if err != nil {
		return fmt.Errorf("find link: %w", err)
	}

	want := map[string]*vnl.Addr{}
	for _, cidr := range wantCIDRs {
		if dhcpTokens[cidr] {
			continue
		}

		ip, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}

		ipNet.IP = ip
		want[ipNet.String()] = &vnl.Addr{IPNet: ipNet}
		if noDAD && ip.To4() == nil {
			want[ipNet.String()].Flags = unix.IFA_F_NODAD
		}
	}

	actual, err := vnl.AddrList(link, vnl.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("addr list %s: %w", dev, err)
	}

	actualSet := make(map[string]bool, len(actual))
	for _, a := range actual {
		actualSet[a.IPNet.String()] = true
	}

	for _, a := range actual {
		cidr := a.IPNet.String()
		if want[cidr] != nil {
			continue
		}

		if keepAddr(a, dev) {
			continue
		}

		if dryRun {
			log.Info().Str("dev", dev).Str("addr", cidr).Msg("would remove addr")
			continue
		}

		log.Info().Str("dev", dev).Str("addr", cidr).Msg("remove addr")
		ac := a
		if err := addrDel(link, &ac); err != nil {
			log.Warn().Err(err).Str("dev", dev).Str("addr", cidr).Msg("remove addr")
		}
	}

	for cidr, a := range want {
		if actualSet[cidr] {
			continue
		}

		if dryRun {
			log.Info().Str("dev", dev).Str("addr", cidr).Msg("would add addr")
			continue
		}

		log.Info().Str("dev", dev).Str("addr", cidr).Msg("add addr")
		if err := addrAdd(link, a); err != nil {
			log.Warn().Err(err).Str("dev", dev).Str("addr", cidr).Msg("add addr")
		}
	}

	return nil
}

func keepAddr(a vnl.Addr, dev string) bool {
	if a.Protocol == rtprotKeepalived {
		return true
	}

	if a.Flags&unix.IFA_F_PERMANENT == 0 {
		return true
	}

	if a.IP.IsLinkLocalUnicast() {
		return true
	}

	if dev == "lo" && a.IP.IsLoopback() {
		return true
	}

	return false
}

func addWantedAddrs(wantByDev map[string][]string, dev string, addrs []string) {
	if dev == "" {
		return
	}

	wantByDev[dev] = append(wantByDev[dev], addrs...)
}
