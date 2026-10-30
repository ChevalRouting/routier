//go:build linux

package netlink

import (
	"fmt"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	vnl "github.com/vishvananda/netlink"
)

func addVNetLinks(cfg *config.Config, out map[string]linkSpec) {
	for name, iface := range cfg.Interfaces {
		if iface.Type != "vxlan-bridge" {
			continue
		}

		svd := config.VNetLinkName(name, "x")
		out[svd] = linkSpec{kind: kindVXLAN, device: svd, mtu: iface.MTU, vxlan: config.InterfaceVXLAN(iface)}
		out[iface.Device] = linkSpec{kind: kindBridge, device: iface.Device, mtu: iface.MTU, members: []string{svd}}
	}

	for name, iface := range cfg.Interfaces {
		if iface.Type != "vnet" || iface.VNet == nil {
			continue
		}

		parent := cfg.Interfaces[iface.Select]
		if parent == nil {
			continue
		}

		mtu := iface.MTU
		if mtu == 0 {
			mtu = parent.MTU
		}

		access, peer := config.VNetLinkName(name, "a"), config.VNetLinkName(name, "b")
		out[access] = linkSpec{kind: kindVeth, device: access, peer: peer, mtu: mtu}
		out[peer] = linkSpec{kind: kindVeth, device: peer, mtu: mtu}
		members := []string{peer}
		if iface.Bridge != nil {
			members = append(members, iface.Bridge.MemberDevices...)
		}

		out[iface.Device] = linkSpec{kind: kindBridge, device: iface.Device, mtu: mtu, members: members, vrf: iface.VRF, preserveUnmanaged: true}

		if iface.Bridge != nil {
			spec := out[iface.Device]
			spec.stp = iface.Bridge.STP
			out[iface.Device] = spec
		}

		transport := out[parent.Device]
		transport.members = append(transport.members, access)
		out[parent.Device] = transport
		if iface.VNet.Gateway != nil {
			gateway := config.VNetLinkName(name, "g")
			out[gateway] = linkSpec{kind: kindMacvlan, device: gateway, parent: iface.Device, mac: config.VNetGatewayMAC(iface.VNet), mtu: mtu, vrf: iface.VRF}
		}
	}

	for name, vrf := range cfg.VRFs {
		if vrf.L3VNI == nil {
			continue
		}

		transport := cfg.Interfaces[vrf.L3VNI.Transport]
		if transport == nil {
			continue
		}

		svi := config.L3VNILinkName(name)
		out[svi] = linkSpec{kind: kindVLAN, device: svi, parent: transport.Device, vlanID: vrf.L3VNI.VLAN, vrf: name, mtu: transport.MTU}
	}
}

func ensureVNetPair(spec linkSpec, actual map[string]vnl.Link, dryRun bool) error {
	a, aOK := actual[spec.device]
	b, bOK := actual[spec.peer]
	if aOK || bOK {
		if !aOK || !bOK || a.Type() != "veth" || b.Type() != "veth" || a.Attrs().Alias != managedAlias || b.Attrs().Alias != managedAlias {
			return fmt.Errorf("vnet pair %s/%s conflicts with existing links", spec.device, spec.peer)
		}

		index, err := vnl.VethPeerIndex(a.(*vnl.Veth))
		if err != nil {
			return fmt.Errorf("vnet pair %s/%s: read peer: %w", spec.device, spec.peer, err)
		}

		if index != b.Attrs().Index {
			return fmt.Errorf("vnet pair %s/%s has an unexpected peer index %d", spec.device, spec.peer, index)
		}

		return nil
	}

	if dryRun {
		log.Info().Str("link", spec.device).Str("peer", spec.peer).Msg("would create vnet access pair")
		return nil
	}

	if err := linkAdd(&vnl.Veth{LinkAttrs: vnl.LinkAttrs{Name: spec.device, Alias: managedAlias, MTU: spec.mtu}, PeerName: spec.peer}); err != nil {
		return fmt.Errorf("create vnet pair: %w", err)
	}

	for _, name := range []string{spec.device, spec.peer} {
		link, err := vnl.LinkByName(name)
		if err != nil {
			return err
		}

		if err := linkSetAlias(link, managedAlias); err != nil {
			return err
		}

		actual[name] = link
	}

	return nil
}

func reconcileVNetMappings(cfg *config.Config, actual map[string]vnl.Link, dryRun bool) error {
	for name, iface := range cfg.Interfaces {
		if iface.Type != "vxlan-bridge" {
			continue
		}

		if dryRun {
			log.Info().Str("bridge", iface.Device).Msg("would reconcile vnet vlan-to-vni mappings")
			continue
		}

		br, vx := actual[iface.Device], actual[config.VNetLinkName(name, "x")]
		if br == nil || vx == nil || br.Type() != "bridge" {
			return fmt.Errorf("vxlan-bridge %s has missing or invalid links", name)
		}

		if err := setVNetBridge(br); err != nil {
			return fmt.Errorf("enable vlan filtering: %w", err)
		}

		if err := vnl.LinkSetLearning(vx, false); err != nil {
			return fmt.Errorf("disable svd learning: %w", err)
		}

		if err := vnl.LinkSetVlanTunnel(vx, true); err != nil {
			return fmt.Errorf("enable vlan tunnel: %w", err)
		}

		want := map[uint16]uint32{}
		for vname, vn := range cfg.Interfaces {
			if vn.Type != "vnet" || vn.Select != name || vn.VNet == nil {
				continue
			}

			vid, vni := uint16(vn.VNet.VLAN), uint32(vn.VNet.VNI)
			want[vid] = vni
			access := actual[config.VNetLinkName(vname, "a")]
			if access == nil {
				return fmt.Errorf("vnet %s access port missing", vname)
			}

			if err := configureVNetAccess(access, vid); err != nil {
				return fmt.Errorf("vnet %s access: %w", vname, err)
			}
		}

		for _, vrf := range cfg.VRFs {
			if vrf.L3VNI == nil || vrf.L3VNI.Transport != name {
				continue
			}

			vid := uint16(vrf.L3VNI.VLAN)
			want[vid] = uint32(vrf.L3VNI.VNI)
			if err := vnl.BridgeVlanAdd(br, vid, false, false, true, false); err != nil {
				return fmt.Errorf("vxlan-bridge %s l3vni vlan %d: %w", name, vid, err)
			}
		}

		if err := reconcileVNITunnels(vx, want); err != nil {
			return fmt.Errorf("vxlan-bridge %s mappings: %w", name, err)
		}
	}

	return nil
}

func configureVNetAccess(link vnl.Link, vid uint16) error {
	if err := vnl.LinkSetLearning(link, true); err != nil {
		return err
	}

	vlans, err := vnl.BridgeVlanList()
	if err != nil {
		return err
	}

	for _, vlan := range vlans[int32(link.Attrs().Index)] {
		if vlan.Vid != vid {
			if err := vnl.BridgeVlanDel(link, vlan.Vid, false, false, false, false); err != nil {
				return err
			}
		}
	}

	return vnl.BridgeVlanAdd(link, vid, true, true, false, false)
}

func reconcileVNITunnels(vx vnl.Link, want map[uint16]uint32) error {
	tunnels, err := vnl.BridgeVlanTunnelShowDev(vx)
	if err != nil {
		return err
	}

	existing := map[uint16]uint32{}
	for _, tunnel := range tunnels {
		if want[tunnel.Vid] != tunnel.TunId {
			if err := vnl.BridgeVlanDelTunnelInfo(vx, tunnel.Vid, tunnel.TunId, false, false); err != nil {
				return err
			}
		} else {
			existing[tunnel.Vid] = tunnel.TunId
		}
	}

	vlans, err := vnl.BridgeVlanList()
	if err != nil {
		return err
	}

	for _, vlan := range vlans[int32(vx.Attrs().Index)] {
		if _, keep := want[vlan.Vid]; !keep {
			if err := vnl.BridgeVlanDel(vx, vlan.Vid, false, false, false, false); err != nil {
				return err
			}
		}
	}

	for vid, vni := range want {
		if err := vnl.BridgeVlanAdd(vx, vid, false, false, false, false); err != nil {
			return err
		}

		if existing[vid] != vni {
			if err := vnl.BridgeVlanAddTunnelInfo(vx, vid, vni, false, false); err != nil {
				return err
			}
		}
	}

	return nil
}

func setVNetBridge(link vnl.Link) error {
	filtering := true
	pvid := uint16(0)
	return linkModify(&vnl.Bridge{LinkAttrs: vnl.LinkAttrs{Name: link.Attrs().Name, Index: link.Attrs().Index}, VlanFiltering: &filtering, VlanDefaultPVID: &pvid})
}

func validateVNetOwnership(cfg *config.Config, actual map[string]vnl.Link) error {
	for name, iface := range cfg.Interfaces {
		roles := []string{}
		if iface.Type == "vxlan-bridge" {
			roles = []string{"x"}
		}

		if iface.Type == "vnet" {
			roles = []string{"a", "b"}
		}

		for _, role := range roles {
			dev := config.VNetLinkName(name, role)
			if link := actual[dev]; link != nil && link.Attrs().Alias != managedAlias {
				return fmt.Errorf("generated vnet device %s already exists and is not managed by Routier", dev)
			}
		}
	}

	for name, vrf := range cfg.VRFs {
		dev := config.L3VNILinkName(name)
		if link := actual[dev]; vrf.L3VNI != nil && link != nil && link.Attrs().Alias != managedAlias {
			return fmt.Errorf("generated l3vni device %s already exists and is not managed by Routier", dev)
		}
	}

	return nil
}
