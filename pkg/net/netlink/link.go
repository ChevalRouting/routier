//go:build linux

package netlink

import (
	"fmt"
	"net"
	"os"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
	vnl "github.com/vishvananda/netlink"
)

const managedAlias = "routier"

type linkKind int

const (
	kindPhysical linkKind = iota
	kindBridge
	kindBond
	kindVLAN
	kindVXLAN
	kindTunnel
	kindWireguard
	kindVRF
	kindDummy
)

type linkSpec struct {
	kind     linkKind
	device   string
	mtu      int
	members  []string
	stp      bool
	bond     *config.Bond
	parent   string
	vlanID   int
	vxlan    *config.VXLAN
	tunnel   *config.Tunnel
	vrfTable int
	vrf      string
}

func buildDesiredLinks(cfg *config.Config) map[string]linkSpec {
	out := map[string]linkSpec{}
	for name, vrf := range cfg.VRFs {
		out[name] = linkSpec{kind: kindVRF, device: name, vrfTable: vrf.Table}
	}

	for _, iface := range cfg.Interfaces {
		spec := linkSpec{kind: kindPhysical, device: iface.Device, mtu: iface.MTU, vrf: iface.VRF}
		switch iface.Type {
		case "dummy":
			spec.kind = kindDummy
		case "bridge":
			spec.kind = kindBridge
			if iface.Bridge != nil {
				spec.members = iface.Bridge.MemberDevices
				spec.stp = iface.Bridge.STP
			}
		case "vlan":
			spec.kind = kindVLAN
			if parent, ok := cfg.Interfaces[iface.Select]; ok {
				spec.parent = parent.Device
			}

			if iface.VLAN != nil {
				spec.vlanID = iface.VLAN.ID
			}
		case "vxlan":
			spec.kind = kindVXLAN
			spec.vxlan = iface.VXLAN
		case "bond":
			spec.kind = kindBond
			spec.bond = iface.Bond
			if iface.Bond != nil {
				spec.members = iface.Bond.MemberDevices
			}
		}

		out[iface.Device] = spec
	}

	for name, t := range cfg.Tunnels {
		out[name] = linkSpec{kind: kindTunnel, device: name, mtu: t.MTU, tunnel: t}
	}

	for name, wg := range cfg.Wireguard {
		out[name] = linkSpec{kind: kindWireguard, device: name, mtu: wg.MTU}
	}

	return out
}

func LinkExists(name string) bool {
	_, err := vnl.LinkByName(name)
	return err == nil
}

func ManagedWireguardLinks() []string {
	all, err := vnl.LinkList()
	if err != nil {
		return nil
	}

	var out []string
	for _, l := range all {
		if linkKindOf(l) != kindWireguard {
			continue
		}

		if l.Attrs().Alias != managedAlias {
			continue
		}

		out = append(out, l.Attrs().Name)
	}

	return out
}

func reconcileLinks(cfg *config.Config, dryRun bool) error {
	return ReconcileLinks(cfg, dryRun)
}

func ReconcileLinks(cfg *config.Config, dryRun bool) error {
	all, err := vnl.LinkList()
	if err != nil {
		return fmt.Errorf("netlink link list: %w", err)
	}

	actual := make(map[string]vnl.Link, len(all))
	for _, l := range all {
		actual[l.Attrs().Name] = l
	}

	desired := buildDesiredLinks(cfg)

	for _, pass := range []linkKind{kindVLAN, kindVXLAN, kindBridge, kindBond, kindTunnel, kindVRF, kindDummy} {
		for _, l := range all {
			attrs := l.Attrs()
			if attrs.Alias != managedAlias {
				continue
			}

			if _, keep := desired[attrs.Name]; keep {
				continue
			}

			lk := linkKindOf(l)
			if lk < 0 || lk != pass {
				continue
			}

			if dryRun {
				log.Info().Str("link", attrs.Name).Msg("would delete stale link")
				continue
			}

			log.Info().Str("link", attrs.Name).Msg("delete stale link")
			if err := linkDel(l); err != nil {
				return fmt.Errorf("delete stale link %s: %w", attrs.Name, err)
			}

			delete(actual, attrs.Name)
		}
	}

	for _, l := range all {
		attrs := l.Attrs()
		if attrs.Alias != managedAlias || !isPhysical(l) {
			continue
		}

		if _, keep := desired[attrs.Name]; keep {
			continue
		}

		cleanupRemovedInterface(l, dryRun)
	}

	for _, spec := range desired {
		if spec.kind == kindVRF {
			if err := ensureVRF(spec, actual, dryRun); err != nil {
				log.Error().Err(err).Str("link", spec.device).Msg("vrf reconcile")
			}
		}
	}

	for _, spec := range desired {
		if spec.kind == kindDummy {
			if err := ensureDummy(spec, actual, dryRun); err != nil {
				log.Error().Err(err).Str("link", spec.device).Msg("dummy reconcile")
			}
		}
	}

	for _, spec := range desired {
		if spec.kind == kindVXLAN {
			if err := ensureVXLAN(spec, actual, dryRun); err != nil {
				return fmt.Errorf("vxlan reconcile %s: %w", spec.device, err)
			}
		}
	}

	for _, spec := range desired {
		if spec.kind == kindBridge {
			if err := ensureBridge(spec, actual, all, dryRun); err != nil {
				log.Error().Err(err).Str("link", spec.device).Msg("bridge reconcile")
			}
		}
	}

	for _, spec := range desired {
		if spec.kind == kindBond {
			if err := ensureBond(spec, actual, dryRun); err != nil {
				return fmt.Errorf("bond reconcile %s: %w", spec.device, err)
			}
		}
	}

	for _, spec := range desired {
		if spec.kind == kindVLAN {
			if err := ensureVLAN(spec, actual, dryRun); err != nil {
				log.Error().Err(err).Str("link", spec.device).Msg("vlan reconcile")
			}
		}
	}

	for _, spec := range desired {
		if spec.kind == kindTunnel {
			if err := ensureTunnel(spec, actual, dryRun); err != nil {
				log.Error().Err(err).Str("link", spec.device).Msg("tunnel reconcile")
			}
		}
	}

	if !dryRun {
		for _, spec := range desired {
			if spec.kind != kindWireguard {
				continue
			}

			l, ok := actual[spec.device]
			if !ok || l.Attrs().Alias == managedAlias {
				continue
			}

			_ = linkSetAlias(l, managedAlias)
		}
	}

	all, err = vnl.LinkList()
	if err != nil {
		return fmt.Errorf("netlink link list: %w", err)
	}

	for _, l := range all {
		actual[l.Attrs().Name] = l
	}

	for _, spec := range desired {
		l, ok := actual[spec.device]
		if !ok {
			continue
		}

		if spec.kind == kindPhysical && isPhysical(l) && l.Attrs().Alias != managedAlias && !dryRun {
			if err := linkSetAlias(l, managedAlias); err != nil {
				log.Warn().Err(err).Str("link", spec.device).Msg("mark interface managed")
			}
		}

		if spec.mtu > 0 && l.Attrs().MTU != spec.mtu {
			if dryRun {
				log.Info().Str("link", spec.device).Int("mtu", spec.mtu).Msg("would set MTU")
			} else if err := linkSetMTU(l, spec.mtu); err != nil {
				log.Warn().Err(err).Str("link", spec.device).Msg("set MTU")
			}
		}
	}

	if err := bringUpLinks(desired, actual, dryRun, linkSetUp); err != nil {
		return err
	}

	for _, spec := range desired {
		ifaceLink, ok := actual[spec.device]
		if !ok {
			continue
		}

		masterIdx := ifaceLink.Attrs().MasterIndex
		if spec.vrf == "" {
			if masterIdx == 0 {
				continue
			}

			masterLink, err := vnl.LinkByIndex(masterIdx)
			if err != nil || masterLink == nil {
				continue
			}

			if _, isVRF := masterLink.(*vnl.Vrf); !isVRF {
				continue
			}

			if dryRun {
				log.Info().Str("iface", spec.device).Msg("would remove from vrf")
			} else {
				log.Info().Str("iface", spec.device).Msg("remove from vrf")
				if err := linkSetNoMaster(ifaceLink); err != nil {
					log.Warn().Err(err).Str("iface", spec.device).Msg("remove from vrf")
				}
			}

			continue
		}

		vrfLink, ok := actual[spec.vrf]
		if !ok {
			log.Warn().Str("iface", spec.device).Str("vrf", spec.vrf).Msg("vrf device not found, skipping enslave")
			continue
		}

		if masterIdx == vrfLink.Attrs().Index {
			continue
		}

		if dryRun {
			log.Info().Str("iface", spec.device).Str("vrf", spec.vrf).Msg("would enslave to vrf")
		} else {
			log.Info().Str("iface", spec.device).Str("vrf", spec.vrf).Msg("enslave to vrf")
			if err := linkSetMaster(ifaceLink, vrfLink); err != nil {
				log.Warn().Err(err).Str("iface", spec.device).Str("vrf", spec.vrf).Msg("enslave to vrf")
			}
		}
	}

	return nil
}

func cleanupRemovedInterface(l vnl.Link, dryRun bool) {
	dev := l.Attrs().Name
	if dryRun {
		log.Info().Str("link", dev).Msg("would reset removed interface")
		return
	}

	log.Info().Str("link", dev).Msg("reset removed interface")

	if err := syncAddrs(dev, nil, false); err != nil {
		log.Warn().Err(err).Str("dev", dev).Msg("flush removed interface addresses")
	}

	if l.Attrs().MasterIndex != 0 {
		_ = linkSetNoMaster(l)
	}

	resetSLAAC(dev)
	_ = linkSetDown(l)
	_ = linkSetAlias(l, "")
}

func resetSLAAC(dev string) {
	for _, path := range []string{
		fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/accept_ra", dev),
		fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/autoconf", dev),
	} {
		if err := os.WriteFile(path, []byte("0\n"), 0644); err != nil {
			log.Debug().Err(err).Str("path", path).Msg("reset slaac sysctl")
		}
	}
}

func linkKindOf(l vnl.Link) linkKind {
	switch l.(type) {
	case *vnl.Vlan:
		return kindVLAN
	case *vnl.Vxlan:
		return kindVXLAN
	case *vnl.Bridge:
		return kindBridge
	case *vnl.Bond:
		return kindBond
	case *vnl.Iptun, *vnl.Gretun, *vnl.Gretap, *vnl.Sittun, *vnl.Vti, *vnl.Ip6tnl:
		return kindTunnel
	case *vnl.Wireguard:
		return kindWireguard
	case *vnl.Vrf:
		return kindVRF
	case *vnl.Dummy:
		return kindDummy
	}

	return -1
}

func ensureDummy(spec linkSpec, actual map[string]vnl.Link, dryRun bool) error {
	if _, exists := actual[spec.device]; exists {
		return nil
	}

	if dryRun {
		log.Info().Str("link", spec.device).Msg("would create dummy")
		return nil
	}

	log.Info().Str("link", spec.device).Msg("create dummy")
	if err := linkAdd(&vnl.Dummy{LinkAttrs: vnl.LinkAttrs{Name: spec.device}}); err != nil {
		return fmt.Errorf("create dummy %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = linkSetAlias(created, managedAlias)
	_ = linkSetUp(created)
	actual[spec.device] = created
	return nil
}

func ensureVRF(spec linkSpec, actual map[string]vnl.Link, dryRun bool) error {
	if existing, exists := actual[spec.device]; exists {
		if vrf, ok := existing.(*vnl.Vrf); ok && int(vrf.Table) != spec.vrfTable {
			log.Error().
				Str("vrf", spec.device).
				Int("want_table", spec.vrfTable).
				Int("actual_table", int(vrf.Table)).
				Msg("vrf table ID mismatch, delete and recreate the VRF device to apply the new table")
		}

		return nil
	}

	if dryRun {
		log.Info().Str("link", spec.device).Int("table", spec.vrfTable).Msg("would create vrf")
		return nil
	}

	log.Info().Str("link", spec.device).Int("table", spec.vrfTable).Msg("create vrf")

	vl := &vnl.Vrf{
		LinkAttrs: vnl.LinkAttrs{Name: spec.device},
		Table:     uint32(spec.vrfTable),
	}
	if err := linkAdd(vl); err != nil {
		return fmt.Errorf("create vrf %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = linkSetAlias(created, managedAlias)
	_ = linkSetUp(created)
	actual[spec.device] = created
	return nil
}

func ensureBridge(spec linkSpec, actual map[string]vnl.Link, all []vnl.Link, dryRun bool) error {
	bridgeLink, exists := actual[spec.device]
	if !exists {
		if dryRun {
			log.Info().Str("link", spec.device).Msg("would create bridge")
		} else {
			log.Info().Str("link", spec.device).Msg("create bridge")
			if err := linkAdd(&vnl.Bridge{LinkAttrs: vnl.LinkAttrs{Name: spec.device}}); err != nil {
				return fmt.Errorf("create bridge %s: %w", spec.device, err)
			}

			created, err := vnl.LinkByName(spec.device)
			if err != nil {
				return err
			}

			_ = linkSetAlias(created, managedAlias)
			bridgeLink = created
			actual[spec.device] = created
		}
	}

	if !dryRun && spec.stp {
		if err := os.WriteFile(fmt.Sprintf("/sys/class/net/%s/bridge/stp_state", spec.device), []byte("1\n"), 0644); err != nil {
			log.Warn().Err(err).Str("dev", spec.device).Msg("set STP state")
		}
	}

	wantMembers := make(map[string]bool, len(spec.members))
	for _, m := range spec.members {
		wantMembers[m] = true
	}

	if bridgeLink != nil {
		bridgeIdx := bridgeLink.Attrs().Index
		for _, l := range all {
			if l.Attrs().MasterIndex != bridgeIdx {
				continue
			}

			if wantMembers[l.Attrs().Name] {
				continue
			}

			if dryRun {
				log.Info().Str("member", l.Attrs().Name).Str("bridge", spec.device).Msg("would remove bridge member")
			} else {
				_ = linkSetNoMaster(l)
			}
		}
	}

	for _, m := range spec.members {
		ml, ok := actual[m]
		if !ok {
			continue
		}

		if bridgeLink != nil && ml.Attrs().MasterIndex == bridgeLink.Attrs().Index {
			continue
		}

		if dryRun {
			log.Info().Str("member", m).Str("bridge", spec.device).Msg("would add bridge member")
		} else {
			_ = linkSetMaster(ml, bridgeLink)
			_ = linkSetUp(ml)
		}
	}

	return nil
}

func desiredBond(spec linkSpec) *vnl.Bond {
	b := vnl.NewLinkBond(vnl.LinkAttrs{Name: spec.device, Alias: managedAlias})

	mode := spec.bond.Mode
	if mode == "" {
		mode = "balance-rr"
	}

	b.Mode = vnl.StringToBondMode(mode)
	b.Miimon = spec.bond.MIIMon
	if spec.bond.MIIMon > 0 {
		b.UpDelay = spec.bond.UpDelay
		b.DownDelay = spec.bond.DownDelay
	}

	if spec.bond.XmitHashPolicy != "" {
		b.XmitHashPolicy = vnl.StringToBondXmitHashPolicy(spec.bond.XmitHashPolicy)
	}

	if mode == "802.3ad" {
		if spec.bond.LACPRate != "" {
			b.LacpRate = vnl.StringToBondLacpRate(spec.bond.LACPRate)
		}

		if spec.bond.MinLinks > 0 {
			b.MinLinks = spec.bond.MinLinks
		}
	}

	return b
}

func bondChanged(existing vnl.Link, want *vnl.Bond) bool {
	current, ok := existing.(*vnl.Bond)
	if !ok {
		return true
	}

	if current.Mode != want.Mode || current.Miimon != want.Miimon {
		return true
	}

	if want.UpDelay >= 0 && current.UpDelay != want.UpDelay {
		return true
	}

	if want.DownDelay >= 0 && current.DownDelay != want.DownDelay {
		return true
	}

	if want.XmitHashPolicy >= 0 && current.XmitHashPolicy != want.XmitHashPolicy {
		return true
	}

	if want.LacpRate >= 0 && current.LacpRate != want.LacpRate {
		return true
	}

	if want.MinLinks >= 0 && current.MinLinks != want.MinLinks {
		return true
	}

	return false
}

func ensureBond(spec linkSpec, actual map[string]vnl.Link, dryRun bool) error {
	if spec.bond == nil {
		return fmt.Errorf("bond settings missing for %s", spec.device)
	}

	want := desiredBond(spec)

	bondLink, exists := actual[spec.device]
	if exists && bondChanged(bondLink, want) {
		if dryRun {
			log.Info().Str("link", spec.device).Str("mode", want.Mode.String()).Msg("would update bond")
			return nil
		}

		log.Info().Str("link", spec.device).Str("mode", want.Mode.String()).Msg("update bond")
		if err := linkDel(bondLink); err != nil {
			return fmt.Errorf("delete bond %s for recreation: %w", spec.device, err)
		}

		delete(actual, spec.device)
		exists = false
	}

	if !exists {
		if dryRun {
			log.Info().Str("link", spec.device).Str("mode", want.Mode.String()).Msg("would create bond")
			return nil
		}

		log.Info().Str("link", spec.device).Str("mode", want.Mode.String()).Msg("create bond")
		if err := linkAdd(want); err != nil {
			return fmt.Errorf("create bond %s: %w", spec.device, err)
		}

		created, err := vnl.LinkByName(spec.device)
		if err != nil {
			return err
		}

		_ = linkSetAlias(created, managedAlias)
		bondLink = created
		actual[spec.device] = created
	}

	wantMembers := make(map[string]bool, len(spec.members))
	for _, m := range spec.members {
		wantMembers[m] = true
	}

	bondIdx := bondLink.Attrs().Index
	for _, m := range spec.members {
		member, ok := actual[m]
		if !ok {
			return fmt.Errorf("bond %s member %s not found", spec.device, m)
		}

		if member.Attrs().Index == bondIdx {
			return fmt.Errorf("bond %s cannot be its own member (check interface selectors)", spec.device)
		}
	}

	for _, l := range actual {
		if l.Attrs().MasterIndex != bondIdx || wantMembers[l.Attrs().Name] {
			continue
		}

		if dryRun {
			log.Info().Str("member", l.Attrs().Name).Str("bond", spec.device).Msg("would remove bond member")
		} else {
			if err := linkSetNoMaster(l); err != nil {
				return fmt.Errorf("remove bond member %s: %w", l.Attrs().Name, err)
			}
		}
	}

	for _, m := range spec.members {
		ml := actual[m]
		if ml.Attrs().MasterIndex == bondIdx {
			continue
		}

		if dryRun {
			log.Info().Str("member", m).Str("bond", spec.device).Msg("would add bond member")
			continue
		}

		wasUp := ml.Attrs().Flags&net.FlagUp != 0
		if err := linkSetDown(ml); err != nil {
			return fmt.Errorf("bring down bond member %s: %w", m, err)
		}

		if err := linkSetMaster(ml, bondLink); err != nil {
			if wasUp {
				_ = linkSetUp(ml)
			}

			return fmt.Errorf("attach bond member %s to %s: %w", m, spec.device, err)
		}

		if err := linkSetUp(ml); err != nil {
			return fmt.Errorf("bring up bond member %s: %w", m, err)
		}
	}

	if spec.bond.PrimaryDevice != "" && !dryRun {
		path := fmt.Sprintf("/sys/class/net/%s/bonding/primary", spec.device)
		if err := os.WriteFile(path, []byte(spec.bond.PrimaryDevice+"\n"), 0644); err != nil {
			log.Warn().Err(err).Str("bond", spec.device).Str("primary", spec.bond.PrimaryDevice).Msg("set bond primary")
		}
	}

	return nil
}

func ensureVXLAN(spec linkSpec, actual map[string]vnl.Link, dryRun bool) error {
	if spec.vxlan == nil {
		return fmt.Errorf("vxlan settings missing for %s", spec.device)
	}

	vtepIndex := 0
	if spec.vxlan.VTEP != "" {
		vtepName := spec.vxlan.VTEP
		if iface, ok := actual[vtepName]; ok {
			vtepIndex = iface.Attrs().Index
		} else {
			return fmt.Errorf("vtep interface %s not found for vxlan %s", vtepName, spec.device)
		}
	}

	port := spec.vxlan.Port
	if port == 0 {
		port = 4789
	}

	learning := true
	if spec.vxlan.Learning != nil {
		learning = *spec.vxlan.Learning
	}

	group := spec.vxlan.Group
	if group == "" {
		group = spec.vxlan.Remote
	}

	vni := spec.vxlan.VNI
	if spec.vxlan.External {
		vni = 0
	}

	want := &vnl.Vxlan{
		LinkAttrs:    vnl.LinkAttrs{Name: spec.device, Alias: managedAlias},
		VxlanId:      vni,
		VtepDevIndex: vtepIndex,
		SrcAddr:      net.ParseIP(spec.vxlan.Local),
		Group:        net.ParseIP(group),
		Learning:     learning,
		Port:         port,
		FlowBased:    spec.vxlan.External,
		VniFilter:    spec.vxlan.VNIFilter,
	}

	if existing, exists := actual[spec.device]; exists {
		if !vxlanChanged(existing, want) {
			return markManagedVXLAN(existing, dryRun, linkSetAlias)
		}

		if dryRun {
			log.Info().Str("link", spec.device).Int("vni", spec.vxlan.VNI).Msg("would update vxlan")
			return nil
		}

		if err := linkDel(existing); err != nil {
			return fmt.Errorf("delete vxlan %s for recreation: %w", spec.device, err)
		}
	}

	if dryRun {
		log.Info().Str("link", spec.device).Int("vni", spec.vxlan.VNI).Msg("would create vxlan")
		return nil
	}

	log.Info().Str("link", spec.device).Int("vni", spec.vxlan.VNI).Msg("create vxlan")
	if err := linkAdd(want); err != nil {
		return fmt.Errorf("create vxlan %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	actual[spec.device] = created
	return nil
}

func markManagedVXLAN(link vnl.Link, dryRun bool, setAlias func(vnl.Link, string) error) error {
	if link.Attrs().Alias == managedAlias {
		return nil
	}

	if dryRun {
		log.Info().Str("link", link.Attrs().Name).Msg("would mark vxlan as managed")
		return nil
	}

	if err := setAlias(link, managedAlias); err != nil {
		return fmt.Errorf("mark vxlan %s as managed: %w", link.Attrs().Name, err)
	}

	return nil
}

func vxlanChanged(existing vnl.Link, want *vnl.Vxlan) bool {
	current, ok := existing.(*vnl.Vxlan)
	if !ok {
		return true
	}

	return current.VxlanId != want.VxlanId ||
		current.VtepDevIndex != want.VtepDevIndex ||
		!gwEqual(current.SrcAddr, want.SrcAddr) ||
		!gwEqual(current.Group, want.Group) ||
		current.Learning != want.Learning || current.Port != want.Port ||
		current.FlowBased != want.FlowBased || current.VniFilter != want.VniFilter
}

func ensureVLAN(spec linkSpec, actual map[string]vnl.Link, dryRun bool) error {
	if _, exists := actual[spec.device]; exists {
		return nil
	}

	parent, ok := actual[spec.parent]
	if !ok {
		return fmt.Errorf("parent %s not found for vlan %s", spec.parent, spec.device)
	}

	if dryRun {
		log.Info().Str("link", spec.device).Int("id", spec.vlanID).Msg("would create vlan")
		return nil
	}

	log.Info().Str("link", spec.device).Int("id", spec.vlanID).Msg("create vlan")
	vl := &vnl.Vlan{
		LinkAttrs: vnl.LinkAttrs{Name: spec.device, ParentIndex: parent.Attrs().Index},
		VlanId:    spec.vlanID,
	}

	if err := linkAdd(vl); err != nil {
		return fmt.Errorf("create vlan %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = linkSetAlias(created, managedAlias)
	actual[spec.device] = created
	return nil
}

func ensureTunnel(spec linkSpec, actual map[string]vnl.Link, dryRun bool) error {
	t := spec.tunnel

	want, err := tunnelLink(spec.device, t)
	if err != nil {
		return err
	}

	if existing, exists := actual[spec.device]; exists {
		if !tunnelChanged(existing, t) {
			return nil
		}

		if dryRun {
			log.Info().Str("link", spec.device).Msg("would update tunnel")
			return nil
		}

		log.Info().Str("link", spec.device).Msg("update tunnel")
		if err := linkModify(want); err != nil {
			log.Warn().Err(err).Str("link", spec.device).Msg("modify tunnel, recreating")
			_ = linkDel(existing)
			if err := linkAdd(want); err != nil {
				return fmt.Errorf("recreate tunnel %s: %w", spec.device, err)
			}
		}

		return nil
	}

	if dryRun {
		log.Info().Str("link", spec.device).Str("mode", t.Mode).Msg("would create tunnel")
		return nil
	}

	log.Info().Str("link", spec.device).Str("mode", t.Mode).Msg("create tunnel")
	if err := linkAdd(want); err != nil {
		return fmt.Errorf("create tunnel %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = linkSetAlias(created, managedAlias)
	actual[spec.device] = created
	return nil
}

func tunnelLink(name string, t *config.Tunnel) (vnl.Link, error) {
	attrs := vnl.LinkAttrs{Name: name}
	var local, remote net.IP
	if t.Local != "" {
		local = net.ParseIP(t.Local)
	}

	if t.Remote != "" {
		remote = net.ParseIP(t.Remote)
	}

	ttl := uint8(t.TTL)

	switch t.Mode {
	case "ipip":
		l := &vnl.Iptun{LinkAttrs: attrs, Ttl: ttl}
		if local != nil {
			l.Local = local.To4()
		}

		if remote != nil {
			l.Remote = remote.To4()
		}

		return l, nil
	case "gre":
		l := &vnl.Gretun{LinkAttrs: attrs, Ttl: ttl}
		if local != nil {
			l.Local = local.To4()
		}

		if remote != nil {
			l.Remote = remote.To4()
		}

		return l, nil
	case "ip6gre":
		l := &vnl.Gretun{LinkAttrs: attrs, Ttl: ttl}
		if local != nil {
			l.Local = local
		}

		if remote != nil {
			l.Remote = remote
		}

		return l, nil
	case "gretap":
		l := &vnl.Gretap{LinkAttrs: attrs, Ttl: ttl}
		if local != nil {
			l.Local = local.To4()
		}

		if remote != nil {
			l.Remote = remote.To4()
		}

		return l, nil
	case "sit":
		l := &vnl.Sittun{LinkAttrs: attrs, Ttl: ttl}
		if local != nil {
			l.Local = local.To4()
		}

		if remote != nil {
			l.Remote = remote.To4()
		}

		return l, nil
	case "vti":
		l := &vnl.Vti{LinkAttrs: attrs}
		if local != nil {
			l.Local = local.To4()
		}

		if remote != nil {
			l.Remote = remote.To4()
		}

		return l, nil
	case "ip6tnl", "ip6ip6":
		l := &vnl.Ip6tnl{LinkAttrs: attrs, Ttl: ttl}
		if local != nil {
			l.Local = local
		}

		if remote != nil {
			l.Remote = remote
		}

		return l, nil
	default:
		return nil, fmt.Errorf("unsupported tunnel mode %q", t.Mode)
	}
}

type typer interface {
	Type() string
}

func tunnelChanged(existing vnl.Link, t *config.Tunnel) bool {
	if tv, ok := existing.(typer); ok && tv.Type() != t.Mode {
		return true
	}

	var local, remote net.IP
	if t.Local != "" {
		local = net.ParseIP(t.Local)
	}

	if t.Remote != "" {
		remote = net.ParseIP(t.Remote)
	}

	ttl := uint8(t.TTL)

	switch e := existing.(type) {
	case *vnl.Iptun:
		return !gwEqual(e.Local, local) || !gwEqual(e.Remote, remote) || e.Ttl != ttl
	case *vnl.Gretun:
		return !gwEqual(e.Local, local) || !gwEqual(e.Remote, remote) || e.Ttl != ttl
	case *vnl.Gretap:
		return !gwEqual(e.Local, local) || !gwEqual(e.Remote, remote) || e.Ttl != ttl
	case *vnl.Sittun:
		return !gwEqual(e.Local, local) || !gwEqual(e.Remote, remote) || e.Ttl != ttl
	case *vnl.Vti:
		return !gwEqual(e.Local, local) || !gwEqual(e.Remote, remote)
	case *vnl.Ip6tnl:
		return !gwEqual(e.Local, local) || !gwEqual(e.Remote, remote)
	default:
		return true
	}
}

func bringUpLinks(desired map[string]linkSpec, actual map[string]vnl.Link, dryRun bool, up func(vnl.Link) error) error {
	state := map[string]int{}
	var visit func(string) error
	visit = func(name string) error { return bringUpLinksCallback(desired, actual, dryRun, up, state, visit, name) }
	for name := range desired {
		if err := visit(name); err != nil {
			return err
		}
	}

	return nil
}

func bringUpLinksCallback(desired map[string]linkSpec, actual map[string]vnl.Link, dryRun bool, up func(vnl.Link) error, state map[string]int, visit func(string) error, name string) error {
	if state[name] == 2 {
		return nil
	}

	if state[name] == 1 {
		return fmt.Errorf("link dependency cycle at %s", name)
	}

	state[name] = 1
	spec := desired[name]
	if _, managed := desired[spec.parent]; managed && spec.parent != "" {
		if err := visit(spec.parent); err != nil {
			return err
		}
	}

	if l, ok := actual[name]; ok && l.Attrs().Flags&net.FlagUp == 0 {
		if dryRun {
			log.Info().Str("link", name).Msg("would bring up")
		} else if err := up(l); err != nil {
			return fmt.Errorf("bring up link %s: %w", name, err)
		}
	}

	state[name] = 2
	return nil
}
