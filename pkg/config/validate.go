package config

import (
	"fmt"
	"net"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func isValidIP(s string) bool {
	return net.ParseIP(s) != nil
}

func isPositiveInt(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0
}

func isValidCIDR(s string) bool {
	_, _, err := net.ParseCIDR(s)
	return err == nil
}

var dhcpTokens = map[string]bool{
	"dhcp": true, "dhcp4": true, "dhcp6": true, "slaac": true,
}

func isValidAddress(s string) bool {
	return dhcpTokens[s] || isValidCIDR(s)
}

func isValidIPOrCIDR(s string) bool {
	return isValidIP(s) || isValidCIDR(s)
}

func isValidMTU(mtu int) bool {
	return mtu >= 576 && mtu <= 9000
}

func isValidPort(port int) bool {
	return port >= 1 && port <= 65535
}

type addfunc func(string, ...any)

func Validate(cfg *Config, resolveIfaces bool) []error {
	var errs []error
	add := func(msg string, args ...any) {
		errs = append(errs, fmt.Errorf(msg, args...))
	}

	if cfg.Hostname == "" {
		add("hostname is required")
	}

	validateVRFs(cfg, add)
	validateInterfaces(cfg, add)
	if resolveIfaces {
		resolveInterfaces(cfg, add)
	}

	validateRouting(cfg, add)
	validateTunnels(cfg, add)
	validateWireguard(cfg, add)
	validateNftables(cfg, add)
	validateFriends(cfg, add)
	validateUsers(cfg, add)
	validateServices(cfg, add)
	validateDNS(cfg, add)
	validateLogging(cfg, add)
	validateSSH(cfg, add)
	validateDHCP(cfg, add)

	return errs
}

func validateInterfaces(cfg *Config, add addfunc) {
	vrrpIDs := map[int]string{}
	vlanNames := map[string]string{}

	for name, iface := range cfg.Interfaces {
		if iface.Select == "" && iface.Type != "dummy" && iface.Type != "bridge" {
			add("interfaces.%s: select is required", name)
		}

		if iface.VRF != "" {
			if _, ok := cfg.VRFs[iface.VRF]; !ok {
				add("interfaces.%s: vrf %q not declared in vrfs", name, iface.VRF)
			}
		}

		for i, addr := range iface.Addresses {
			if !isValidAddress(addr) {
				add("interfaces.%s.addresses[%d]: %q is not a valid CIDR or DHCP token", name, i, addr)
			}
		}

		if iface.MTU != 0 && !isValidMTU(iface.MTU) {
			add("interfaces.%s: mtu must be between 576 and 9000", name)
		}

		if iface.Bridge != nil {
			if len(iface.Bridge.Members) == 0 {
				add("interfaces.%s.bridge: at least one member is required", name)
			}

			for _, member := range iface.Bridge.Members {
				if _, ok := cfg.Interfaces[member]; !ok {
					add("interfaces.%s.bridge: member %q not found in interfaces", name, member)
				}
			}
		}

		for i, v := range iface.VRRP {
			if v.ID < 1 || v.ID > 255 {
				add("interfaces.%s.vrrp[%d]: id must be 1-255", name, i)
			} else if prev, ok := vrrpIDs[v.ID]; ok {
				add("interfaces.%s.vrrp[%d]: id %d already used by %s", name, i, v.ID, prev)
			} else {
				vrrpIDs[v.ID] = fmt.Sprintf("interfaces.%s", name)
			}

			if len(v.VIPs) == 0 {
				add("interfaces.%s.vrrp[%d]: at least one vip is required", name, i)
			}

			for j, vip := range v.VIPs {
				if !isValidIPOrCIDR(vip) {
					add("interfaces.%s.vrrp[%d].vips[%d]: %q is not a valid IP or CIDR", name, i, j, vip)
				}
			}

			if v.Priority != 0 && (v.Priority < 1 || v.Priority > 254) {
				add("interfaces.%s.vrrp[%d]: priority must be 1-254", name, i)
			}

			if len(v.Password) > 8 {
				add("interfaces.%s.vrrp[%d]: password must be 8 characters or fewer", name, i)
			}

			if v.Interface != "" {
				if _, ok := cfg.Interfaces[v.Interface]; !ok {
					add("interfaces.%s.vrrp[%d]: interface %q not found in interfaces", name, i, v.Interface)
				}
			}
		}

		for vname, vlan := range iface.VLANs {
			key := fmt.Sprintf("interfaces.%s.vlans.%s", name, vname)
			if prev, ok := vlanNames[vname]; ok {
				add("%s: vlan name %q already used by %s", key, vname, prev)
			} else {
				vlanNames[vname] = key
			}

			if vlan.ID == 0 {
				add("interfaces.%s.vlans.%s: id is required", name, vname)
			}

			for i, addr := range vlan.Addresses {
				if !isValidCIDR(addr) {
					add("interfaces.%s.vlans.%s.addresses[%d]: %q is not a valid CIDR", name, vname, i, addr)
				}
			}

			if vlan.MTU != 0 && !isValidMTU(vlan.MTU) {
				add("interfaces.%s.vlans.%s: mtu must be between 576 and 9000", name, vname)
			}
		}
	}
}

func validateVRFs(cfg *Config, add addfunc) {
	tables := map[int]string{}
	for name, vrf := range cfg.VRFs {
		if vrf.Table <= 0 {
			add("vrfs.%s: table must be a positive integer", name)
		} else if prev, ok := tables[vrf.Table]; ok {
			add("vrfs.%s: table %d already used by vrf %q", name, vrf.Table, prev)
		} else {
			tables[vrf.Table] = name
		}
	}
}

func validateRouting(cfg *Config, add addfunc) {
	r := cfg.Routing
	if r == nil {
		return
	}

	vrfNames := make(map[string]bool, len(cfg.VRFs))
	for name := range cfg.VRFs {
		vrfNames[name] = true
	}

	bfdProfiles := map[string]bool{}
	if r.BFD != nil {
		for _, p := range r.BFD.Profiles {
			bfdProfiles[p.Name] = true
		}
	}

	validateStaticRoutes(r.Static, "routing.static", add)

	if r.BGP != nil {
		validateBGP(r.BGP, "routing.bgp", vrfNames, bfdProfiles, add)
	}

	if r.BFD != nil {
		validateBFD(r.BFD, "routing.bfd", add)
	}

	if r.OSPF != nil {
		validateOSPF(r.OSPF, "routing.ospf", add)
	}

	if r.OSPF6 != nil {
		validateOSPF6(r.OSPF6, "routing.ospf6", add)
	}

	if r.Anycast != nil {
		validateAnycast(r.Anycast, r.BGP, add)
	}

	if r.RADVD != nil {
		validateRADVD(r.RADVD, cfg.Interfaces, add)
	}

	if r.PBR != nil {
		validatePBR(r.PBR, add)
	}

	for vrfName, vrfRouting := range r.VRFs {
		if _, ok := cfg.VRFs[vrfName]; !ok {
			add("routing.vrfs.%s: vrf %q not declared in vrfs", vrfName, vrfName)
		}

		path := fmt.Sprintf("routing.vrfs.%s", vrfName)
		validateStaticRoutes(vrfRouting.Static, path+".static", add)
		if vrfRouting.BGP != nil {
			validateBGP(vrfRouting.BGP, path+".bgp", vrfNames, bfdProfiles, add)
		}

		if vrfRouting.OSPF != nil {
			validateOSPF(vrfRouting.OSPF, path+".ospf", add)
		}

		if vrfRouting.OSPF6 != nil {
			validateOSPF6(vrfRouting.OSPF6, path+".ospf6", add)
		}
	}
}

func validateStaticRoutes(routes []StaticRoute, path string, add addfunc) {
	seen := map[string]bool{}
	for i, s := range routes {
		if s.Destination == "" {
			add("%s[%d]: destination is required", path, i)
		} else if !isValidCIDR(s.Destination) {
			add("%s[%d]: destination %q is not a valid CIDR", path, i, s.Destination)
		} else {
			key := s.Destination
			if s.Via != "" {
				key += " via " + s.Via
			}

			if seen[key] {
				add("%s[%d]: duplicate route %q", path, i, key)
			}

			seen[key] = true
		}

		if s.Via == "" && s.Dev == "" {
			add("%s[%d]: via or dev is required", path, i)
		}

		if s.Via != "" && !isValidIP(s.Via) {
			add("%s[%d]: via %q is not a valid IP", path, i, s.Via)
		}
	}
}

func validateBFD(bfd *BFD, path string, add addfunc) {
	seen := map[string]bool{}
	for i, p := range bfd.Profiles {
		if p.Name == "" {
			add("%s.profiles[%d]: name is required", path, i)
		} else if seen[p.Name] {
			add("%s.profiles[%d]: duplicate profile name %q", path, i, p.Name)
		}

		seen[p.Name] = true
		if p.DetectMultiplier < 0 {
			add("%s.profiles[%d]: detect_multiplier must be positive", path, i)
		}

		if p.ReceiveInterval < 0 {
			add("%s.profiles[%d]: receive_interval must be positive", path, i)
		}

		if p.TransmitInterval < 0 {
			add("%s.profiles[%d]: transmit_interval must be positive", path, i)
		}

		if p.MinimumTTL < 0 || p.MinimumTTL > 254 {
			add("%s.profiles[%d]: minimum_ttl must be between 0 and 254", path, i)
		}
	}
}

func validateBGP(bgp *BGP, path string, vrfNames map[string]bool, bfdProfiles map[string]bool, add addfunc) {
	if bgp.ASN == 0 {
		add("%s: asn is required", path)
	}

	if bgp.RouterID == "" {
		add("%s: router_id is required", path)
	} else if !isValidIP(bgp.RouterID) {
		add("%s: router_id %q is not a valid IP", path, bgp.RouterID)
	}

	for plName, entries := range bgp.PrefixLists {
		seen := map[int]int{}
		for i, e := range entries {
			if prev, dup := seen[e.Seq]; dup {
				add("%s.prefix_lists.%s[%d]: duplicate seq %d (also used by entry %d)", path, plName, i, e.Seq, prev)
			} else {
				seen[e.Seq] = i
			}

			if e.Action != "" && !slices.Contains([]string{"permit", "deny"}, e.Action) {
				add("%s.prefix_lists.%s[%d]: action must be permit or deny", path, plName, i)
			}

			if e.Prefix != "" && !isValidCIDR(e.Prefix) {
				add("%s.prefix_lists.%s[%d]: prefix %q is not a valid CIDR", path, plName, i, e.Prefix)
			}
		}
	}

	for rmName, entries := range bgp.RouteMaps {
		seen := map[int]int{}
		for i, e := range entries {
			if prev, dup := seen[e.Seq]; dup {
				add("%s.route_maps.%s[%d]: duplicate seq %d (also used by entry %d)", path, rmName, i, e.Seq, prev)
			} else {
				seen[e.Seq] = i
			}

			if e.Action != "" && !slices.Contains([]string{"permit", "deny"}, e.Action) {
				add("%s.route_maps.%s[%d]: action must be permit or deny", path, rmName, i)
			}

			if e.Call != "" {
				if _, ok := bgp.RouteMaps[e.Call]; !ok {
					add("%s.route_maps.%s[%d]: call %q not found", path, rmName, i, e.Call)
				}
			}

			if e.OnMatch != "" && e.OnMatch != "next" {
				if goto_, ok := strings.CutPrefix(e.OnMatch, "goto "); !ok || !isPositiveInt(goto_) {
					add("%s.route_maps.%s[%d]: on_match must be \"next\" or \"goto <seq>\"", path, rmName, i)
				}
			}
		}
	}

	for af, afConfig := range bgp.AddressFamilies {
		if afConfig == nil {
			continue
		}

		for i, vrfName := range afConfig.ImportVRF {
			if vrfName != "default" && vrfName != "main" && !vrfNames[vrfName] {
				add("%s.address_families.%s.import_vrf[%d]: vrf %q not declared in vrfs", path, af, i, vrfName)
			}
		}
	}

	for i, n := range bgp.Neighbors {
		if n.Address == "" {
			add("%s.neighbors[%d]: address required", path, i)
		} else if !isValidIP(n.Address) {
			add("%s.neighbors[%d]: address %q is not a valid IP", path, i, n.Address)
		}

		if n.RemoteASN == 0 {
			add("%s.neighbors[%d]: remote_asn required", path, i)
		}

		if n.EBGPMultihop < 0 {
			add("%s.neighbors[%d]: ebgp_multihop need to be positive", path, i)
		}

		if n.BFDProfile != "" {
			if !n.BFD {
				add("%s.neighbors[%d]: bfd_profile set but bfd is not enabled", path, i)
			}

			if !bfdProfiles[n.BFDProfile] {
				add("%s.neighbors[%d]: bfd_profile %q not found in routing.bfd.profiles", path, i, n.BFDProfile)
			}
		}

		for af, nafConfig := range n.AddressFamilies {
			if nafConfig.PrefixListIn != "" {
				if _, ok := bgp.PrefixLists[nafConfig.PrefixListIn]; !ok {
					add("%s.neighbors[%d].address_families.%s: prefix_list_in %q not found", path, i, af, nafConfig.PrefixListIn)
				}
			}

			if nafConfig.PrefixListOut != "" {
				if _, ok := bgp.PrefixLists[nafConfig.PrefixListOut]; !ok {
					add("%s.neighbors[%d].address_families.%s: prefix_list_out %q not found", path, i, af, nafConfig.PrefixListOut)
				}
			}

			if nafConfig.RouteMapIn != "" {
				if _, ok := bgp.RouteMaps[nafConfig.RouteMapIn]; !ok {
					add("%s.neighbors[%d].address_families.%s: route_map_in %q not found", path, i, af, nafConfig.RouteMapIn)
				}
			}

			if nafConfig.RouteMapOut != "" {
				if _, ok := bgp.RouteMaps[nafConfig.RouteMapOut]; !ok {
					add("%s.neighbors[%d].address_families.%s: route_map_out %q not found", path, i, af, nafConfig.RouteMapOut)
				}
			}
		}
	}
}

func validateOSPF(ospf *OSPF, path string, add addfunc) {
	if ospf.RouterID == "" {
		add("%s: router_id is required", path)
	} else if !isValidIP(ospf.RouterID) {
		add("%s: router_id %q is not a valid IP", path, ospf.RouterID)
	}

	for i, area := range ospf.Areas {
		if area.Type != "" && !slices.Contains([]string{"normal", "stub", "nssa"}, area.Type) {
			add("%s.areas[%d]: type must be normal, stub, or nssa", path, i)
		}

		for j, network := range area.Networks {
			if !isValidCIDR(network) {
				add("%s.areas[%d].networks[%d]: %q is not a valid CIDR", path, i, j, network)
			}
		}
	}
}

func validateOSPF6(ospf6 *OSPF6, path string, add addfunc) {
	if ospf6.RouterID == "" {
		add("%s: router_id is required", path)
	} else if !isValidIP(ospf6.RouterID) {
		add("%s: router_id %q is not a valid IP", path, ospf6.RouterID)
	}

	for i, area := range ospf6.Areas {
		if area.Type != "" && !slices.Contains([]string{"stub", "nssa"}, area.Type) {
			add("%s.areas[%d]: type must be stub or nssa", path, i)
		}

		for j, r := range area.Ranges {
			if !isValidCIDR(r) {
				add("%s.areas[%d].ranges[%d]: %q is not a valid CIDR", path, i, j, r)
			}
		}
	}
}

func validateAnycast(anycast *AnycastConfig, bgp *BGP, add addfunc) {
	if bgp == nil || bgp.ASN == 0 || bgp.RouterID == "" {
		add("routing.anycast: requires routing.bgp with valid asn and router_id")
	}

	for i, s := range anycast.Services {
		if s.Name == "" {
			add("routing.anycast.services[%d]: name is required", i)
		}

		if len(s.AnycastIPs) == 0 {
			add("routing.anycast.services[%d]: at least one anycast_ip is required", i)
		}

		for j, ip := range s.AnycastIPs {
			if !isValidIPOrCIDR(ip) {
				add("routing.anycast.services[%d].anycast_ips[%d]: %q is not a valid IP or CIDR", i, j, ip)
			}
		}

		for j, e := range s.Endpoints {
			if e.IP == "" {
				add("routing.anycast.services[%d].endpoints[%d]: ip is required", i, j)
			} else if !isValidIP(e.IP) {
				add("routing.anycast.services[%d].endpoints[%d]: ip %q is not a valid IP", i, j, e.IP)
			}

			if e.HTTPCheck == nil && e.DNSCheck == nil {
				add("routing.anycast.services[%d].endpoints[%d]: http_check or dns_check is required", i, j)
			}

			if e.HTTPCheck != nil && e.DNSCheck != nil {
				add("routing.anycast.services[%d].endpoints[%d]: http_check and dns_check are mutually exclusive", i, j)
			}

			if e.HTTPCheck != nil && e.HTTPCheck.ExpectedCode != 0 {
				if e.HTTPCheck.ExpectedCode < 100 || e.HTTPCheck.ExpectedCode > 599 {
					add("routing.anycast.services[%d].endpoints[%d].http_check: expected_code must be a valid HTTP status (100-599)", i, j)
				}
			}

			if e.DNSCheck != nil && e.DNSCheck.Resolver != "" && !isValidIP(e.DNSCheck.Resolver) {
				add("routing.anycast.services[%d].endpoints[%d].dns_check: resolver %q is not a valid IP", i, j, e.DNSCheck.Resolver)
			}
		}
	}
}

func isKnownInterface(name string, ifaces map[string]*Interface) bool {
	if _, ok := ifaces[name]; ok {
		return true
	}

	for _, iface := range ifaces {
		if _, ok := iface.VLANs[name]; ok {
			return true
		}
	}

	return false
}

func validateRADVD(radvd *RADVDConfig, ifaces map[string]*Interface, add addfunc) {
	preferences := []string{"low", "medium", "high"}
	for ifName, riface := range radvd.Interfaces {
		if !isKnownInterface(ifName, ifaces) {
			add("routing.radvd.interfaces.%s: not found in interfaces or vlans", ifName)
		}

		if riface.AdvDefaultPreference != "" && !slices.Contains(preferences, riface.AdvDefaultPreference) {
			add("routing.radvd.interfaces.%s: adv_default_preference must be low, medium, or high", ifName)
		}

		if riface.MinRtrAdvInterval > 0 && riface.MaxRtrAdvInterval > 0 && riface.MinRtrAdvInterval >= riface.MaxRtrAdvInterval {
			add("routing.radvd.interfaces.%s: min_rtr_adv_interval must be less than max_rtr_adv_interval", ifName)
		}

		for i, p := range riface.Prefixes {
			if !isValidCIDR(p.Prefix) {
				add("routing.radvd.interfaces.%s.prefixes[%d]: prefix %q is not a valid CIDR", ifName, i, p.Prefix)
			}
		}

		for i, route := range riface.Routes {
			if !isValidCIDR(route.Prefix) {
				add("routing.radvd.interfaces.%s.routes[%d]: prefix %q is not a valid CIDR", ifName, i, route.Prefix)
			}

			if route.Preference != "" && !slices.Contains(preferences, route.Preference) {
				add("routing.radvd.interfaces.%s.routes[%d]: preference must be low, medium, or high", ifName, i)
			}
		}

		if riface.RDNSS != nil {
			for i, srv := range riface.RDNSS.Servers {
				if !isValidIP(srv) {
					add("routing.radvd.interfaces.%s.rdnss.servers[%d]: %q is not a valid IP", ifName, i, srv)
				}
			}
		}
	}
}

func validatePBR(pbr *PBR, add addfunc) {
	for name, group := range pbr.NexthopGroups {
		if len(group.Nexthops) == 0 {
			add("routing.pbr.nexthop_groups.%s: at least one nexthop is required", name)
		}

		for i, nh := range group.Nexthops {
			if nh.Address == "" && nh.Dev == "" {
				add("routing.pbr.nexthop_groups.%s.nexthops[%d]: address or interface (dev) is required", name, i)
			}
		}
	}

	for name, entries := range pbr.Maps {
		seen := map[int]int{}
		for i, e := range entries {
			if e.Seq <= 0 {
				add("routing.pbr.maps.%s[%d]: seq must be a positive integer", name, i)
			} else if prev, dup := seen[e.Seq]; dup {
				add("routing.pbr.maps.%s[%d]: duplicate seq %d (also used by entry %d)", name, i, e.Seq, prev)
			} else {
				seen[e.Seq] = i
			}

			if e.MatchSrc != "" && !isValidCIDR(e.MatchSrc) {
				add("routing.pbr.maps.%s[%d]: match_src must be a valid CIDR (e.g. 10.0.0.0/8)", name, i)
			}

			if e.MatchDst != "" && !isValidCIDR(e.MatchDst) {
				add("routing.pbr.maps.%s[%d]: match_dst must be a valid CIDR (e.g. 10.0.0.0/8)", name, i)
			}

			if e.SetNexthopGroup == "" && e.SetNexthop == "" {
				add("routing.pbr.maps.%s[%d]: set_nexthop_group or set_nexthop is required", name, i)
			}

			if e.SetNexthopGroup != "" && e.SetNexthop != "" {
				add("routing.pbr.maps.%s[%d]: set_nexthop_group and set_nexthop are mutually exclusive", name, i)
			}

			if e.SetNexthopGroup != "" {
				if _, ok := pbr.NexthopGroups[e.SetNexthopGroup]; !ok {
					add("routing.pbr.maps.%s[%d]: set_nexthop_group %q not found in nexthop_groups", name, i, e.SetNexthopGroup)
				}
			}
		}
	}

	for iface, mapName := range pbr.Policies {
		if _, ok := pbr.Maps[mapName]; !ok {
			add("routing.pbr.policies.%s: map %q not found in pbr.maps", iface, mapName)
		}
	}
}

func validateTunnels(cfg *Config, add addfunc) {
	tunnelModes := []string{"gre", "gretap", "sit", "vti", "ipip", "ip6tnl", "ip6ip6", "ip6gre"}
	for name, t := range cfg.Tunnels {
		if t.Mode == "" {
			add("tunnels.%s: mode is required", name)
		} else if !slices.Contains(tunnelModes, t.Mode) {
			add("tunnels.%s: mode %q must be one of: %s", name, t.Mode, strings.Join(tunnelModes, ", "))
		}

		if t.Local == "" && t.Remote == "" {
			add("tunnels.%s: at least one of local or remote is required", name)
		}

		if t.Local != "" && !isValidIP(t.Local) {
			add("tunnels.%s: local %q is not a valid IP", name, t.Local)
		}

		if t.Remote != "" && !isValidIP(t.Remote) {
			add("tunnels.%s: remote %q is not a valid IP", name, t.Remote)
		}

		for i, addr := range t.Addresses {
			if !isValidCIDR(addr) {
				add("tunnels.%s.addresses[%d]: %q is not a valid CIDR", name, i, addr)
			}
		}

		if t.MTU != 0 && !isValidMTU(t.MTU) {
			add("tunnels.%s: mtu must be between 576 and 9000", name)
		}
	}
}

func validateWireguard(cfg *Config, add addfunc) {
	friendNames := map[string]bool{}
	for _, f := range cfg.Friends {
		friendNames[f.Name] = true
	}

	for name, wg := range cfg.Wireguard {
		if wg.Friend != "" && !friendNames[wg.Friend] {
			add("wireguard.%s.friend: %q does not reference an existing friend", name, wg.Friend)
		}

		if wg.PrivateKey == "" && wg.PrivateKeyFile == "" {
			add("wireguard.%s: private_key or private_key_file required", name)
		}

		if wg.PrivateKey != "" && wg.PrivateKeyFile != "" {
			add("wireguard.%s: private_key and private_key_file are mutually exclusive", name)
		}

		if wg.ListenPort != 0 && !isValidPort(wg.ListenPort) {
			add("wireguard.%s: listen_port must be between 1 and 65535", name)
		}

		for i, addr := range wg.Addresses {
			if !isValidCIDR(addr) {
				add("wireguard.%s.addresses[%d]: %q is not a valid CIDR", name, i, addr)
			}
		}

		if wg.MTU != 0 && !isValidMTU(wg.MTU) {
			add("wireguard.%s: mtu must be between 576 and 9000", name)
		}

		for i, p := range wg.Peers {
			if p.PublicKey == "" {
				add("wireguard.%s.peers[%d]: public_key required", name, i)
			}

			if p.PresharedKey != "" && p.PresharedKeyFile != "" {
				add("wireguard.%s.peers[%d]: preshared_key and preshared_key_file are mutually exclusive", name, i)
			}

			if p.Endpoint != "" {
				if _, _, err := net.SplitHostPort(p.Endpoint); err != nil {
					add("wireguard.%s.peers[%d]: endpoint %q is not a valid host:port", name, i, p.Endpoint)
				}
			}

			for j, ip := range p.AllowedIPs {
				if !isValidCIDR(ip) {
					add("wireguard.%s.peers[%d].allowed_ips[%d]: %q is not a valid CIDR", name, i, j, ip)
				}
			}

			if p.Keepalive < 0 || p.Keepalive > 65535 {
				add("wireguard.%s.peers[%d]: keepalive must be between 0 and 65535", name, i)
			}
		}
	}
}

func validateNftables(cfg *Config, add addfunc) {
	if cfg.Nftables == nil {
		return
	}

	validActions := []string{"accept", "drop", "reject", "return", "log", "masquerade", "dnat", "snat", "redirect"}
	ownedChains := []string{"input", "forward", "output", "prerouting", "postrouting"}
	validPolicies := []string{"accept", "drop"}

	for name, ch := range cfg.Nftables.Chains {
		if !slices.Contains(ownedChains, name) {
			add("nftables.chains: %q is not a routier-owned chain (one of: %s); use include for custom tables", name, strings.Join(ownedChains, ", "))
			continue
		}

		if ch == nil {
			continue
		}

		if ch.Policy != "" && !slices.Contains(validPolicies, ch.Policy) {
			add("nftables.chains.%s.policy: %q must be one of: %s", name, ch.Policy, strings.Join(validPolicies, ", "))
		}

		for k, rule := range ch.Managed {
			if rule.Action == "" {
				add("nftables.chains.%s.managed[%d]: action is required", name, k)
			} else if !slices.Contains(validActions, rule.Action) {
				add("nftables.chains.%s.managed[%d]: action %q must be one of: %s", name, k, rule.Action, strings.Join(validActions, ", "))
			}
		}
	}
}

func validateFriends(cfg *Config, add addfunc) {
	seen := map[string]bool{}
	for i, f := range cfg.Friends {
		if f.Name == "" {
			add("friends[%d]: name is required", i)
		} else if seen[f.Name] {
			add("friends[%d]: duplicate name %q", i, f.Name)
		}

		seen[f.Name] = true
		if f.URL == "" {
			add("friends[%d]: url is required", i)
		}

		if f.Token == "" {
			add("friends[%d]: token is required", i)
		}

		if fp := f.Identity.Fingerprint; fp != "" && !strings.HasPrefix(fp, "SHA256:") {
			add("friends[%d].identity.fingerprint: must be a SHA256: fingerprint", i)
		}

		if f.HA != nil && f.HA.Enabled && f.HA.Link != nil {
			if f.HA.Link.Interface == "" {
				add("friends[%d].ha.link: interface is required", i)
			}

			if f.HA.Link.Address == "" {
				add("friends[%d].ha.link: address is required", i)
			}
		}

		if f.Sync != nil {
			for j, s := range f.Sync.Sections {
				if !friendSyncSections[s] {
					add("friends[%d].sync.sections[%d]: %q is not replicable (supported: vrrp, conntrackd)", i, j, s)
				}
			}

			for k := range f.Sync.Overrides {
				if k == "" {
					add("friends[%d].sync.overrides: empty override path", i)
				}
			}
		}
	}
}

var friendSyncSections = map[string]bool{
	"vrrp":       true,
	"conntrackd": true,
}

func validateUsers(cfg *Config, add addfunc) {
}

var allowedServiceDestPrefixes = []string{
	"/etc/routier/out/",
	"/etc/",
	"/var/lib/routier/",
}

func validateServiceDest(dest string) bool {
	clean := filepath.Clean(dest)
	for _, prefix := range allowedServiceDestPrefixes {
		if strings.HasPrefix(clean, prefix) {
			return true
		}
	}

	return false
}

func validateServices(cfg *Config, add addfunc) {
	for name, s := range cfg.Services {
		for i, c := range s.Configs {
			if c.Src == "" {
				add("services.%s.configs[%d]: src required", name, i)
			}

			if c.Dest == "" {
				add("services.%s.configs[%d]: dest required", name, i)
			} else if !validateServiceDest(c.Dest) {
				add("services.%s.configs[%d]: dest %q is not in an allowed directory", name, i, c.Dest)
			}

			if c.Mode != 0 && c.Mode > 0o777 {
				add("services.%s.configs[%d]: mode must be a valid Unix permission (0-0777)", name, i)
			}
		}
	}
}

func validateDNS(cfg *Config, add addfunc) {
	if cfg.DNS == nil {
		return
	}

	for i, servIP := range cfg.DNS.Nameservers {
		if !isValidIP(servIP) {
			add("dns.nameservers[%d]: %q is not a valid IPv4 or IPv6 address", i, servIP)
		}
	}
}

func validateSSH(cfg *Config, add addfunc) {
	s := cfg.SSH
	if s == nil {
		return
	}

	if s.Port != 0 && !isValidPort(s.Port) {
		add("ssh: port must be between 1 and 65535")
	}

	yesNo := []string{"yes", "no"}
	if s.PermitRootLogin != "" && !slices.Contains([]string{"yes", "no", "prohibit-password", "forced-commands-only"}, s.PermitRootLogin) {
		add("ssh: permit_root_login must be yes, no, prohibit-password, or forced-commands-only")
	}

	if s.PasswordAuth != "" && !slices.Contains(yesNo, s.PasswordAuth) {
		add("ssh: password_auth must be yes or no")
	}

	if s.PubkeyAuth != "" && !slices.Contains(yesNo, s.PubkeyAuth) {
		add("ssh: pubkey_auth must be yes or no")
	}

	if s.AllowTcpForwarding != "" && !slices.Contains([]string{"yes", "no", "local", "remote"}, s.AllowTcpForwarding) {
		add("ssh: allow_tcp_forwarding must be yes, no, local, or remote")
	}

	if s.X11Forwarding != "" && !slices.Contains(yesNo, s.X11Forwarding) {
		add("ssh: x11_forwarding must be yes or no")
	}

	if s.MaxAuthTries < 0 {
		add("ssh: max_auth_tries must be positive")
	}

	if s.LoginGraceTime < 0 {
		add("ssh: login_grace_time must be positive")
	}

	if s.ClientAliveInterval < 0 {
		add("ssh: client_alive_interval must be positive")
	}

	if s.ClientAliveCountMax < 0 {
		add("ssh: client_alive_count_max must be positive")
	}
}

func validateLogging(cfg *Config, add addfunc) {
	if cfg.Logging == nil {
		return
	}

	if cfg.Logging.Level != "" && !slices.Contains([]string{"debug", "info", "warn", "error"}, cfg.Logging.Level) {
		add("logging: level must be debug, info, warn, or error")
	}

	if cfg.Logging.Target != "" && !slices.Contains([]string{"syslog", "file", "stdout", "stderr"}, cfg.Logging.Target) {
		add("logging: target must be syslog, file, stdout, or stderr")
	}

	if cfg.Logging.Target == "file" && cfg.Logging.File == "" {
		add("logging: file is required when target is file")
	}
}

func isV6(s string) bool {
	return strings.Contains(s, ":")
}

func isValidMAC(s string) bool {
	hw, err := net.ParseMAC(s)
	return err == nil && len(hw) == 6
}

func validateDHCP(cfg *Config, add addfunc) {
	d := cfg.DHCP
	if d == nil || !d.Enabled {
		return
	}

	if d.ControlAgent != nil && d.ControlAgent.URL != "" {
		return
	}

	validateKeaSubnets(d.Subnets4, false, "dhcp.subnets4", cfg, add)
	validateKeaSubnets(d.Subnets6, true, "dhcp.subnets6", cfg, add)
}

func validateKeaSubnets(subnets []KeaSubnet, v6 bool, path string, cfg *Config, add addfunc) {
	fam := "IPv4"
	if v6 {
		fam = "IPv6"
	}

	seen := map[string]bool{}
	for i, s := range subnets {
		p := fmt.Sprintf("%s[%d]", path, i)

		if s.Subnet == "" {
			add("%s: subnet is required", p)
			continue
		}

		_, network, err := net.ParseCIDR(s.Subnet)
		if err != nil {
			add("%s: subnet %q is not a valid CIDR", p, s.Subnet)
			continue
		}

		if isV6(s.Subnet) != v6 {
			add("%s: subnet %q is not an %s network", p, s.Subnet, fam)
			continue
		}

		if seen[network.String()] {
			add("%s: duplicate subnet %q", p, network.String())
		}

		seen[network.String()] = true

		if s.Interface != "" && !isKnownInterface(s.Interface, cfg.Interfaces) {
			add("%s: interface %q not found in interfaces or vlans", p, s.Interface)
		}

		if s.Gateway != "" {
			if v6 {
				add("%s: gateway is not supported for IPv6 subnets", p)
			} else if !isValidIP(s.Gateway) {
				add("%s: gateway %q is not a valid IP", p, s.Gateway)
			} else if !network.Contains(net.ParseIP(s.Gateway)) {
				add("%s: gateway %q is not in subnet %s", p, s.Gateway, network.String())
			}
		}

		for j, dns := range s.DNS {
			if !isValidIP(dns) {
				add("%s.dns[%d]: %q is not a valid IP", p, j, dns)
			}
		}

		for j, pool := range s.Pools {
			validateKeaRange(pool, network, v6, fmt.Sprintf("%s.pools[%d]", p, j), true, add)
		}

		for j, ex := range s.Exclusions {
			validateKeaRange(ex, network, v6, fmt.Sprintf("%s.exclusions[%d]", p, j), false, add)
		}

		for j, res := range s.Reservations {
			validateKeaReservation(res, network, v6, fmt.Sprintf("%s.reservations[%d]", p, j), add)
		}
	}
}

func validateKeaReservation(res KeaReservation, network *net.IPNet, v6 bool, path string, add addfunc) {
	if res.HWAddress == "" && res.DUID == "" {
		add("%s: hw_address or duid is required", path)
	}

	if res.HWAddress != "" && !isValidMAC(res.HWAddress) {
		add("%s: hw_address %q is not a valid MAC", path, res.HWAddress)
	}

	if res.IPAddress == "" {
		return
	}

	ip := net.ParseIP(res.IPAddress)
	if ip == nil {
		add("%s: ip_address %q is not a valid IP", path, res.IPAddress)
	} else if isV6(res.IPAddress) != v6 {
		add("%s: ip_address %q does not match the subnet family", path, res.IPAddress)
	} else if !network.Contains(ip) {
		add("%s: ip_address %q is not in subnet %s", path, res.IPAddress, network.String())
	}
}

func validateKeaRange(spec string, network *net.IPNet, v6 bool, path string, requireRange bool, add addfunc) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		add("%s: is empty", path)
		return
	}

	if start, end, ok := strings.Cut(spec, "-"); ok {
		validateKeaRangeIP(strings.TrimSpace(start), network, v6, path+" start", add)
		validateKeaRangeIP(strings.TrimSpace(end), network, v6, path+" end", add)
		return
	}

	if requireRange {
		add("%s: %q must be a range (start-end)", path, spec)
		return
	}

	validateKeaRangeIP(spec, network, v6, path, add)
}

func validateKeaRangeIP(s string, network *net.IPNet, v6 bool, path string, add addfunc) {
	ip := net.ParseIP(s)
	if ip == nil {
		add("%s: %q is not a valid IP", path, s)
		return
	}

	if isV6(s) != v6 {
		add("%s: %q does not match the subnet family", path, s)
		return
	}

	if !network.Contains(ip) {
		add("%s: %q is not in subnet %s", path, s, network.String())
	}
}

func ValidationSet(cfg *Config) map[string]struct{} {
	set := make(map[string]struct{})
	for _, e := range Validate(cfg, false) {
		set[e.Error()] = struct{}{}
	}

	return set
}

func NewValidationErrors(before map[string]struct{}, cfg *Config) []string {
	var added []string
	for _, e := range Validate(cfg, false) {
		if _, existed := before[e.Error()]; !existed {
			added = append(added, e.Error())
		}
	}

	return added
}
