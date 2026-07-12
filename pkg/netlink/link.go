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
	kindVLAN
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
	parent   string
	vlanID   int
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
		}

		if iface.Bridge != nil {
			spec.kind = kindBridge
			spec.members = iface.Bridge.MemberDevices
			spec.stp = iface.Bridge.STP
		}

		out[iface.Device] = spec
		for _, vlan := range iface.VLANs {
			out[vlan.Device] = linkSpec{
				kind: kindVLAN, device: vlan.Device, mtu: vlan.MTU,
				parent: iface.Device, vlanID: vlan.ID,
			}
		}
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
	all, err := vnl.LinkList()
	if err != nil {
		return fmt.Errorf("netlink link list: %w", err)
	}

	actual := make(map[string]vnl.Link, len(all))
	for _, l := range all {
		actual[l.Attrs().Name] = l
	}

	desired := buildDesiredLinks(cfg)

	for _, pass := range []linkKind{kindVLAN, kindBridge, kindTunnel, kindVRF, kindDummy} {
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
			if err := vnl.LinkDel(l); err != nil {
				log.Warn().Err(err).Str("link", attrs.Name).Msg("delete stale link")
			}
		}
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
		if spec.kind == kindBridge {
			if err := ensureBridge(spec, actual, all, dryRun); err != nil {
				log.Error().Err(err).Str("link", spec.device).Msg("bridge reconcile")
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

			_ = vnl.LinkSetAlias(l, managedAlias)
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

		if spec.mtu > 0 && l.Attrs().MTU != spec.mtu {
			if dryRun {
				log.Info().Str("link", spec.device).Int("mtu", spec.mtu).Msg("would set MTU")
			} else if err := vnl.LinkSetMTU(l, spec.mtu); err != nil {
				log.Warn().Err(err).Str("link", spec.device).Msg("set MTU")
			}
		}

		if l.Attrs().Flags&net.FlagUp == 0 {
			if dryRun {
				log.Info().Str("link", spec.device).Msg("would bring up")
			} else if err := vnl.LinkSetUp(l); err != nil {
				log.Warn().Err(err).Str("link", spec.device).Msg("link up")
			}
		}
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
				if err := vnl.LinkSetNoMaster(ifaceLink); err != nil {
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
			if err := vnl.LinkSetMaster(ifaceLink, vrfLink); err != nil {
				log.Warn().Err(err).Str("iface", spec.device).Str("vrf", spec.vrf).Msg("enslave to vrf")
			}
		}
	}

	return nil
}

func linkKindOf(l vnl.Link) linkKind {
	switch l.(type) {
	case *vnl.Vlan:
		return kindVLAN
	case *vnl.Bridge:
		return kindBridge
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
	if err := vnl.LinkAdd(&vnl.Dummy{LinkAttrs: vnl.LinkAttrs{Name: spec.device}}); err != nil {
		return fmt.Errorf("create dummy %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = vnl.LinkSetAlias(created, managedAlias)
	_ = vnl.LinkSetUp(created)
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
	if err := vnl.LinkAdd(vl); err != nil {
		return fmt.Errorf("create vrf %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = vnl.LinkSetAlias(created, managedAlias)
	_ = vnl.LinkSetUp(created)
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
			if err := vnl.LinkAdd(&vnl.Bridge{LinkAttrs: vnl.LinkAttrs{Name: spec.device}}); err != nil {
				return fmt.Errorf("create bridge %s: %w", spec.device, err)
			}

			created, err := vnl.LinkByName(spec.device)
			if err != nil {
				return err
			}

			_ = vnl.LinkSetAlias(created, managedAlias)
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
				_ = vnl.LinkSetNoMaster(l)
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
			_ = vnl.LinkSetMaster(ml, bridgeLink)
			_ = vnl.LinkSetUp(ml)
		}
	}

	return nil
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

	if err := vnl.LinkAdd(vl); err != nil {
		return fmt.Errorf("create vlan %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = vnl.LinkSetAlias(created, managedAlias)
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
		if err := vnl.LinkModify(want); err != nil {
			log.Warn().Err(err).Str("link", spec.device).Msg("modify tunnel, recreating")
			_ = vnl.LinkDel(existing)
			if err := vnl.LinkAdd(want); err != nil {
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
	if err := vnl.LinkAdd(want); err != nil {
		return fmt.Errorf("create tunnel %s: %w", spec.device, err)
	}

	created, err := vnl.LinkByName(spec.device)
	if err != nil {
		return err
	}

	_ = vnl.LinkSetAlias(created, managedAlias)
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
