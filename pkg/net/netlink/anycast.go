//go:build linux

package netlink

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	vnl "github.com/vishvananda/netlink"
)

func ensureMacvlan(spec linkSpec, actual map[string]vnl.Link, dryRun bool) error {
	parent, ok := actual[spec.parent]
	if !ok {
		if dryRun {
			return nil
		}

		return fmt.Errorf("parent %s not found", spec.parent)
	}

	mac, err := net.ParseMAC(spec.mac)
	if err != nil {
		return err
	}

	if existing, exists := actual[spec.device]; exists {
		if existing.Type() == "macvlan" && existing.Attrs().ParentIndex == parent.Attrs().Index {
			return keepMAC(existing, mac, dryRun)
		}

		if dryRun {
			log.Info().Str("link", spec.device).Msg("would recreate gateway")
			return nil
		}

		if err := linkDel(existing); err != nil {
			return err
		}

		delete(actual, spec.device)
	}

	if dryRun {
		log.Info().Str("link", spec.device).Str("parent", spec.parent).Str("mac", spec.mac).Msg("would create gateway")
		return nil
	}

	log.Info().Str("link", spec.device).Str("parent", spec.parent).Str("mac", spec.mac).Msg("create gateway")
	attrs := vnl.LinkAttrs{Name: spec.device, ParentIndex: parent.Attrs().Index, HardwareAddr: mac, MTU: spec.mtu, Alias: managedAlias}
	if err := linkAdd(&vnl.Macvlan{LinkAttrs: attrs, Mode: vnl.MACVLAN_MODE_PRIVATE}); err != nil {
		return err
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = linkSetAlias(created, managedAlias)
	actual[spec.device] = created
	return nil
}

func keepMAC(link vnl.Link, mac net.HardwareAddr, dryRun bool) error {
	if link.Attrs().HardwareAddr.String() == mac.String() {
		return nil
	}

	if dryRun {
		log.Info().Str("link", link.Attrs().Name).Str("mac", mac.String()).Msg("would set gateway mac")
		return nil
	}

	return vnl.LinkSetHardwareAddr(link, mac)
}

func gatewayAddrs(gw *config.VNetGateway) []string {
	addrs := make([]string, 0, len(gw.IPs))
	for _, ip := range gw.IPs {
		if addr, err := netip.ParseAddr(ip); err == nil {
			addrs = append(addrs, netip.PrefixFrom(addr, addr.BitLen()).String())
		}
	}

	return addrs
}

func keepAnycastGatewaysLocal(cfg *config.Config, root string, dryRun bool) {
	for name, iface := range cfg.Interfaces {
		if iface.Type != "vnet" || iface.VNet == nil || iface.VNet.Gateway == nil {
			continue
		}

		gateway := config.VNetLinkName(name, "g")
		for _, dev := range []string{iface.Device, gateway} {
			ensureSysctl(cfg, root, "net.ipv4.conf."+dev+".arp_ignore", "1", "answer ARP only for the addresses of the interface", dryRun)
		}

		ensureSysctl(cfg, root, "net.ipv4.conf."+gateway+".rp_filter", "2", "accept traffic to the gateway that returns through the bridge", dryRun)
	}
}
