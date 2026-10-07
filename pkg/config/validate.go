package config

import (
	"fmt"
	"net"
	"path/filepath"
	"regexp"
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

func isValidDNSLabel(label string) bool {
	if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}

	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}

	return true
}

func isValidDNSName(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}

	trimmed := strings.TrimSuffix(s, ".")
	if trimmed == "" {
		return false
	}

	for _, label := range strings.Split(trimmed, ".") {
		if !isValidDNSLabel(label) {
			return false
		}
	}

	return true
}

func NormalizeDNSName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}

	return s + "."
}

func isValidZoneRecordName(name, zone string) bool {
	if name == "@" || name == "*" {
		return true
	}

	rel := strings.TrimPrefix(name, "*.")
	if rel == "" {
		return false
	}

	if !strings.HasSuffix(rel, ".") {
		return isValidDNSName(rel)
	}

	if !isValidDNSName(rel) {
		return false
	}

	owner, z := NormalizeDNSName(rel), NormalizeDNSName(zone)

	return owner == z || strings.HasSuffix(owner, "."+z)
}

var dnsCacheSizeRe = regexp.MustCompile(`^\d+[kKmMgG]?$`)

func isValidDNSCacheSize(s string) bool {
	return dnsCacheSizeRe.MatchString(s)
}

const (
	DNSModeForwarder     = "forwarder"
	DNSModeAuthoritative = "authoritative"
	DNSModeBoth          = "both"
)

var dnsModes = []string{DNSModeForwarder, DNSModeAuthoritative, DNSModeBoth}

func (s *DNSServer) ResolvedMode() string {
	if s.Mode != "" {
		return s.Mode
	}

	if len(s.Zones) == 0 {
		return DNSModeForwarder
	}

	if len(s.Upstreams) > 0 || len(s.Forward) > 0 {
		return DNSModeBoth
	}

	return DNSModeAuthoritative
}

func (s *DNSServer) Recurses() bool {
	return s.ResolvedMode() != DNSModeAuthoritative
}

type ListenRefKind int

const (
	ListenLiteral ListenRefKind = iota
	ListenIface
	ListenVIPs
)

type ListenRef struct {
	Kind ListenRefKind
	Name string
}

var listenRefRe = regexp.MustCompile(`^(iface|vips)\(([^()]+)\)$`)

func ParseListenRef(entry string) (ListenRef, bool) {
	entry = strings.TrimSpace(entry)
	if m := listenRefRe.FindStringSubmatch(entry); m != nil {
		kind := ListenIface
		if m[1] == "vips" {
			kind = ListenVIPs
		}

		return ListenRef{Kind: kind, Name: strings.TrimSpace(m[2])}, true
	}

	if isValidIP(entry) {
		return ListenRef{Kind: ListenLiteral, Name: entry}, true
	}

	return ListenRef{}, false
}

func ResolveListen(cfg *Config, entry string) ([]string, error) {
	ref, ok := ParseListenRef(entry)
	if !ok {
		return nil, fmt.Errorf("%q is not an IP address, iface(name), or vips(name)", entry)
	}

	if ref.Kind == ListenLiteral {
		return []string{ref.Name}, nil
	}

	if !isKnownInterface(ref.Name, cfg.Interfaces) {
		return nil, fmt.Errorf("%s: interface %q not found in interfaces", entry, ref.Name)
	}

	if ref.Kind == ListenVIPs {
		vips := hostIPs(vrrpVIPs(cfg, ref.Name))
		if len(vips) == 0 {
			return nil, fmt.Errorf("%s: interface %q has no vrrp vips", entry, ref.Name)
		}

		return vips, nil
	}

	addrs := hostIPs(staticAddresses(cfg, ref.Name))
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%s: interface %q has no static address to listen on", entry, ref.Name)
	}

	return addrs, nil
}

func staticAddresses(cfg *Config, name string) []string {
	if iface, ok := cfg.Interfaces[name]; ok {
		return iface.Addresses
	}

	return nil
}

func vrrpVIPs(cfg *Config, name string) []string {
	if cfg.HA == nil {
		return nil
	}

	var vips []string
	for _, v := range cfg.HA.VRRP {
		if v.Interface == name {
			vips = append(vips, v.VIPs...)
		}
	}

	return vips
}

func hostIPs(addrs []string) []string {
	var out []string
	for _, addr := range addrs {
		if dhcpTokens[addr] {
			continue
		}

		if ip, _, err := net.ParseCIDR(addr); err == nil {
			out = append(out, ip.String())
			continue
		}

		if ip := net.ParseIP(addr); ip != nil {
			out = append(out, ip.String())
		}
	}

	return out
}

const maxIfnameLen = 15

func isValidIfname(s string) bool {
	if s == "" || len(s) > maxIfnameLen || s == "." || s == ".." {
		return false
	}

	return !strings.ContainsAny(s, "/: \t\n\v\f\r")
}

const ifnameRule = "must be at most 15 characters with no '/', ':' or whitespace"

var reservedIfnames = map[string]bool{
	"lo":       true,
	"tunl0":    true,
	"gre0":     true,
	"gretap0":  true,
	"erspan0":  true,
	"sit0":     true,
	"ip6tnl0":  true,
	"ip6gre0":  true,
	"ip_vti0":  true,
	"ip6_vti0": true,
}

func isReservedIfname(s string) bool {
	return reservedIfnames[s]
}

type addfunc func(string, ...any)

func validateDeviceName(prefix, name string, add addfunc) {
	if !isValidIfname(name) {
		add("%s.%s: name is used as the device name and %s", prefix, name, ifnameRule)
	}

	if isReservedIfname(name) {
		add("%s.%s: %q is a reserved kernel device name and cannot be used", prefix, name, name)
	}
}

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
	validateHA(cfg, add)
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
	validateMonitoring(cfg, add)

	return errs
}

var probeTargetPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`)

func validateMonitoring(cfg *Config, add addfunc) {
	if cfg.Monitoring == nil {
		return
	}

	seen := map[string]bool{}
	for index, probe := range cfg.Monitoring.Probes {
		path := fmt.Sprintf("monitoring.probes[%d]", index)
		if strings.TrimSpace(probe.Name) == "" {
			add("%s.name is required", path)
		}

		if seen[probe.Name] {
			add("%s.name %q is duplicated", path, probe.Name)
		}

		seen[probe.Name] = true
		if !probeTargetPattern.MatchString(probe.Target) {
			add("%s.target must be an IP address or hostname", path)
		}

		if probe.Interval != 0 && probe.Interval < 60 {
			add("%s.interval must be at least 60 seconds", path)
		}

		if probe.Timeout != 0 && (probe.Timeout < 100 || probe.Timeout > 55000) {
			add("%s.timeout must be between 100 and 55000 milliseconds", path)
		}
	}

	if lldp := cfg.Monitoring.LLDP; lldp != nil {
		for i, iface := range lldp.Interfaces {
			if strings.TrimSpace(iface) == "" {
				add("monitoring.lldp.interfaces[%d] must not be empty", i)
			}
		}
	}
}

func validateVXLANPorts(cfg *Config, add addfunc) {
	type vxlanDev struct {
		name      string
		external  bool
		vnifilter bool
	}

	byPort := map[int][]vxlanDev{}
	for name, iface := range cfg.Interfaces {
		if iface.Type != "vxlan" || iface.VXLAN == nil {
			continue
		}

		port := iface.VXLAN.Port
		if port == 0 {
			port = 4789
		}

		byPort[port] = append(byPort[port], vxlanDev{name, iface.VXLAN.External, iface.VXLAN.VNIFilter})
	}

	for port, devs := range byPort {
		if len(devs) < 2 {
			continue
		}

		for _, d := range devs {
			if d.external && !d.vnifilter {
				add("interfaces.%s.vxlan: an external vxlan without vnifilter claims every vni on udp port %d, so it cannot share the port with other vxlan interfaces", d.name, port)
			}
		}
	}
}

func validateInterfaces(cfg *Config, add addfunc) {
	validateVXLANPorts(cfg, add)

	for name, iface := range cfg.Interfaces {
		if iface.Select == "" && iface.Type != "dummy" && iface.Type != "bridge" && iface.Type != "vxlan" && iface.Type != "bond" {
			add("interfaces.%s: select is required", name)
		}

		if iface.Type == "dummy" || iface.Type == "bridge" || iface.Type == "vxlan" || iface.Type == "bond" || iface.Type == "vlan" {
			validateDeviceName("interfaces", name, add)
		}

		if iface.Type == "vxlan" {
			validateVXLAN(name, iface.VXLAN, cfg, add)

			if iface.Bridge != nil {
				add("interfaces.%s: vxlan interfaces cannot also be bridges", name)
			}

			if iface.Bond != nil {
				add("interfaces.%s: vxlan interfaces cannot also be bonds", name)
			}
		} else if iface.VXLAN != nil {
			add("interfaces.%s: vxlan settings require type vxlan", name)
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

		if iface.Type == "bond" {
			validateBond(name, iface, cfg, add)
		} else if iface.Bond != nil {
			add("interfaces.%s: bond settings require type bond", name)
		}

		if iface.Type == "vlan" {
			if iface.VLAN == nil {
				add("interfaces.%s: vlan settings are required", name)
			} else if iface.VLAN.ID < 1 || iface.VLAN.ID > 4094 {
				add("interfaces.%s.vlan.id: id must be 1-4094", name)
			}

			if iface.Select != "" {
				if _, ok := cfg.Interfaces[iface.Select]; !ok {
					add("interfaces.%s: parent %q not found in interfaces", name, iface.Select)
				}
			}
		} else if iface.VLAN != nil {
			add("interfaces.%s: vlan settings require type vlan", name)
		}
	}
}

var bondModes = map[string]bool{
	"balance-rr": true, "active-backup": true, "balance-xor": true,
	"broadcast": true, "802.3ad": true, "balance-tlb": true, "balance-alb": true,
}

var bondHashPolicies = map[string]bool{
	"layer2": true, "layer3+4": true, "layer2+3": true,
	"encap2+3": true, "encap3+4": true, "vlan+srcmac": true,
}

var bondLACPRates = map[string]bool{"slow": true, "fast": true}

func validateBond(name string, iface *Interface, cfg *Config, add addfunc) {
	bond := iface.Bond
	if bond == nil {
		add("interfaces.%s: bond settings are required", name)
		return
	}

	if iface.Bridge != nil {
		add("interfaces.%s: bond interfaces cannot also be bridges", name)
	}

	if len(bond.Members) == 0 {
		add("interfaces.%s.bond: at least one member is required", name)
	}

	seen := map[string]bool{}
	for _, member := range bond.Members {
		if _, ok := cfg.Interfaces[member]; !ok {
			add("interfaces.%s.bond: member %q not found in interfaces", name, member)
		}

		if seen[member] {
			add("interfaces.%s.bond: member %q listed more than once", name, member)
		}

		seen[member] = true
	}

	mode := bond.Mode
	if mode == "" {
		mode = "balance-rr"
	}

	if !bondModes[bond.Mode] && bond.Mode != "" {
		add("interfaces.%s.bond: mode %q is not one of balance-rr, active-backup, balance-xor, broadcast, 802.3ad, balance-tlb, balance-alb", name, bond.Mode)
	}

	if bond.XmitHashPolicy != "" {
		if !bondHashPolicies[bond.XmitHashPolicy] {
			add("interfaces.%s.bond: xmit_hash_policy %q is not one of layer2, layer2+3, layer3+4, encap2+3, encap3+4, vlan+srcmac", name, bond.XmitHashPolicy)
		}

		if mode != "balance-xor" && mode != "802.3ad" {
			add("interfaces.%s.bond: xmit_hash_policy only applies to balance-xor and 802.3ad modes", name)
		}
	}

	if bond.LACPRate != "" {
		if !bondLACPRates[bond.LACPRate] {
			add("interfaces.%s.bond: lacp_rate %q is not slow or fast", name, bond.LACPRate)
		}

		if mode != "802.3ad" {
			add("interfaces.%s.bond: lacp_rate only applies to 802.3ad mode", name)
		}
	}

	if bond.MinLinks < 0 {
		add("interfaces.%s.bond: min_links cannot be negative", name)
	}

	if bond.MIIMon < 0 {
		add("interfaces.%s.bond: miimon cannot be negative", name)
	}

	if bond.UpDelay < 0 || bond.DownDelay < 0 {
		add("interfaces.%s.bond: updelay and downdelay cannot be negative", name)
	}

	if (bond.UpDelay > 0 || bond.DownDelay > 0) && bond.MIIMon == 0 {
		add("interfaces.%s.bond: updelay and downdelay require miimon to be set", name)
	}

	if bond.Primary != "" {
		if _, ok := cfg.Interfaces[bond.Primary]; !ok {
			add("interfaces.%s.bond: primary %q not found in interfaces", name, bond.Primary)
		} else if !seen[bond.Primary] {
			add("interfaces.%s.bond: primary %q must also be a member", name, bond.Primary)
		}

		if mode != "active-backup" && mode != "balance-tlb" && mode != "balance-alb" {
			add("interfaces.%s.bond: primary only applies to active-backup, balance-tlb, and balance-alb modes", name)
		}
	}
}

func validateVXLAN(name string, vxlan *VXLAN, cfg *Config, add addfunc) {
	if vxlan == nil {
		add("interfaces.%s: vxlan settings are required", name)
		return
	}

	if vxlan.External {
		if vxlan.VNI != 0 {
			add("interfaces.%s.vxlan: vni is ignored when external is set (the vni comes from tunnel metadata)", name)
		}
	} else {
		if vxlan.VNIFilter {
			add("interfaces.%s.vxlan: vnifilter requires external", name)
		}

		if vxlan.VNI < 1 || vxlan.VNI > 16777215 {
			add("interfaces.%s.vxlan: vni must be between 1 and 16777215", name)
		}
	}

	if vxlan.Local != "" && !isValidIP(vxlan.Local) {
		add("interfaces.%s.vxlan: local %q is not a valid IP", name, vxlan.Local)
	}

	if vxlan.Remote != "" && !isValidIP(vxlan.Remote) {
		add("interfaces.%s.vxlan: remote %q is not a valid IP", name, vxlan.Remote)
	}

	if vxlan.Group != "" {
		ip := net.ParseIP(vxlan.Group)
		if ip == nil || !ip.IsMulticast() {
			add("interfaces.%s.vxlan: group %q is not a multicast IP", name, vxlan.Group)
		}
	}

	if vxlan.Remote != "" && vxlan.Group != "" {
		add("interfaces.%s.vxlan: remote and group are mutually exclusive", name)
	}

	if vxlan.VTEP != "" {
		if _, ok := cfg.Interfaces[vxlan.VTEP]; !ok && !isValidIfname(vxlan.VTEP) {
			add("interfaces.%s.vxlan: vtep %q is not a valid interface", name, vxlan.VTEP)
		}
	}

	if vxlan.Port != 0 && !isValidPort(vxlan.Port) {
		add("interfaces.%s.vxlan: port must be between 1 and 65535", name)
	}
}

func validateVRFs(cfg *Config, add addfunc) {
	tables := map[int]string{}
	for name, vrf := range cfg.VRFs {
		validateDeviceName("vrfs", name, add)

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
		validateBGP(r.BGP, r.BGP, "routing.bgp", vrfNames, bfdProfiles, add)
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
			validateBGP(vrfRouting.BGP, r.BGP, path+".bgp", vrfNames, bfdProfiles, add)
		}

		if vrfRouting.OSPF != nil {
			validateOSPF(vrfRouting.OSPF, path+".ospf", add)
		}

		if vrfRouting.OSPF6 != nil {
			validateOSPF6(vrfRouting.OSPF6, path+".ospf6", add)
		}
	}
}

func validateHA(cfg *Config, add addfunc) {
	if cfg.HA == nil {
		return
	}

	vrrpIDs := map[int]string{}

	for i, v := range cfg.HA.VRRP {
		if v.Interface == "" {
			add("ha.vrrp[%d]: interface is required", i)
		} else if !isKnownInterface(v.Interface, cfg.Interfaces) {
			add("ha.vrrp[%d]: interface %q not found in interfaces", i, v.Interface)
		}

		if v.ID < 1 || v.ID > 255 {
			add("ha.vrrp[%d]: id must be 1-255", i)
		} else if prev, ok := vrrpIDs[v.ID]; ok {
			add("ha.vrrp[%d]: id %d already used by %s", i, v.ID, prev)
		} else {
			vrrpIDs[v.ID] = fmt.Sprintf("ha.vrrp[%d]", i)
		}

		if len(v.VIPs) == 0 {
			add("ha.vrrp[%d]: at least one vip is required", i)
		}

		for j, vip := range v.VIPs {
			if !isValidIPOrCIDR(vip) {
				add("ha.vrrp[%d].vips[%d]: %q is not a valid IP or CIDR", i, j, vip)
			}
		}

		if v.Priority != 0 && (v.Priority < 1 || v.Priority > 254) {
			add("ha.vrrp[%d]: priority must be 1-254", i)
		}

		if len(v.Password) > 8 {
			add("ha.vrrp[%d]: password must be 8 characters or fewer", i)
		}

		if v.Transport != "" && !isKnownInterface(v.Transport, cfg.Interfaces) {
			add("ha.vrrp[%d]: transport %q not found in interfaces", i, v.Transport)
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

func validateBGP(bgp *BGP, global *BGP, path string, vrfNames map[string]bool, bfdProfiles map[string]bool, add addfunc) {
	var routeMaps map[string][]RouteMapEntry
	var prefixLists map[string][]PrefixEntry
	if global != nil {
		routeMaps = global.RouteMaps
		prefixLists = global.PrefixLists
	}

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
				if _, ok := routeMaps[e.Call]; !ok {
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

		hasEVPNOptions := afConfig.AdvertiseAllVNI || afConfig.AdvertiseDefaultGateway || afConfig.AdvertiseSVIIP ||
			len(afConfig.Advertise) > 0 || len(afConfig.RouteTargetImport) > 0 || len(afConfig.RouteTargetExport) > 0
		if af != "l2vpn-evpn" && hasEVPNOptions {
			add("%s.address_families.%s: EVPN options require the l2vpn-evpn address family", path, af)
		}

		hasVPNOptions := afConfig.RD != "" || len(afConfig.RTVPNImport) > 0 || len(afConfig.RTVPNExport) > 0 ||
			afConfig.LabelVPNExportAuto || afConfig.ImportVPN || afConfig.ExportVPN ||
			afConfig.RouteMapVPNImport != "" || afConfig.RouteMapVPNExport != ""
		if af != "ipv4-unicast" && af != "ipv6-unicast" && hasVPNOptions {
			add("%s.address_families.%s: VPN route-leak options require the ipv4-unicast or ipv6-unicast address family", path, af)
		}

		if afConfig.RouteMapVPNImport != "" {
			if _, ok := routeMaps[afConfig.RouteMapVPNImport]; !ok {
				add("%s.address_families.%s: route_map_vpn_import %q not found in route_maps", path, af, afConfig.RouteMapVPNImport)
			}
		}

		if afConfig.RouteMapVPNExport != "" {
			if _, ok := routeMaps[afConfig.RouteMapVPNExport]; !ok {
				add("%s.address_families.%s: route_map_vpn_export %q not found in route_maps", path, af, afConfig.RouteMapVPNExport)
			}
		}

		for i, advertisedAF := range afConfig.Advertise {
			if !slices.Contains([]string{"ipv4-unicast", "ipv6-unicast"}, advertisedAF) {
				add("%s.address_families.%s.advertise[%d]: must be ipv4-unicast or ipv6-unicast", path, af, i)
			}
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
				if _, ok := prefixLists[nafConfig.PrefixListIn]; !ok {
					add("%s.neighbors[%d].address_families.%s: prefix_list_in %q not found", path, i, af, nafConfig.PrefixListIn)
				}
			}

			if nafConfig.PrefixListOut != "" {
				if _, ok := prefixLists[nafConfig.PrefixListOut]; !ok {
					add("%s.neighbors[%d].address_families.%s: prefix_list_out %q not found", path, i, af, nafConfig.PrefixListOut)
				}
			}

			if nafConfig.RouteMapIn != "" {
				if _, ok := routeMaps[nafConfig.RouteMapIn]; !ok {
					add("%s.neighbors[%d].address_families.%s: route_map_in %q not found", path, i, af, nafConfig.RouteMapIn)
				}
			}

			if nafConfig.RouteMapOut != "" {
				if _, ok := routeMaps[nafConfig.RouteMapOut]; !ok {
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

	for name := range ospf.Interfaces {
		if strings.TrimSpace(name) == "" {
			add("%s.interfaces: interface name is required", path)
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

	for name := range ospf6.Interfaces {
		if strings.TrimSpace(name) == "" {
			add("%s.interfaces: interface name is required", path)
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

	return false
}

func validateRADVD(radvd *RADVDConfig, ifaces map[string]*Interface, add addfunc) {
	preferences := []string{"low", "medium", "high"}
	for ifName, riface := range radvd.Interfaces {
		if !isKnownInterface(ifName, ifaces) {
			add("routing.radvd.interfaces.%s: not found in interfaces", ifName)
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
		if strings.TrimSpace(iface) == "" {
			add("routing.pbr.policies: interface name is required")
		}

		if _, ok := pbr.Maps[mapName]; !ok {
			add("routing.pbr.policies.%s: map %q not found in pbr.maps", iface, mapName)
		}
	}
}

func validateTunnels(cfg *Config, add addfunc) {
	tunnelModes := []string{"gre", "gretap", "sit", "vti", "ipip", "ip6tnl", "ip6ip6", "ip6gre"}
	for name, t := range cfg.Tunnels {
		validateDeviceName("tunnels", name, add)

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
		validateDeviceName("wireguard", name, add)

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

	validateDNSServer(cfg, add)
}

func validateDNSMode(s *DNSServer, add addfunc) {
	if s.Mode != "" && !slices.Contains(dnsModes, s.Mode) {
		add("dns.server: mode %q must be one of: %s", s.Mode, strings.Join(dnsModes, ", "))

		return
	}

	forwards := len(s.Upstreams) > 0 || len(s.Forward) > 0

	switch s.Mode {
	case DNSModeAuthoritative:
		if forwards {
			add("dns.server: mode %q serves only its own zones; remove upstreams and forward, or use %q",
				DNSModeAuthoritative, DNSModeBoth)
		}

		if len(s.Zones) == 0 {
			add("dns.server: mode %q requires at least one zone", DNSModeAuthoritative)
		}
	case DNSModeForwarder:
		if len(s.Zones) > 0 {
			add("dns.server: mode %q does not serve zones; remove them or use %q",
				DNSModeForwarder, DNSModeBoth)
		}
	case DNSModeBoth:
		if len(s.Zones) == 0 {
			add("dns.server: mode %q requires at least one zone; use %q", DNSModeBoth, DNSModeForwarder)
		}

		if !forwards {
			add("dns.server: mode %q requires upstreams or forward; use %q", DNSModeBoth, DNSModeAuthoritative)
		}
	}
}

func validateDNSServer(cfg *Config, add addfunc) {
	s := cfg.DNS.Server
	if s == nil || !s.Enabled {
		return
	}

	validateDNSMode(s, add)

	if len(s.Listen) == 0 {
		add("dns.server: at least one listen address is required")
	}

	for i, entry := range s.Listen {
		if _, err := ResolveListen(cfg, entry); err != nil {
			add("dns.server.listen[%d]: %v", i, err)
		}
	}

	if s.Port != 0 && !isValidPort(s.Port) {
		add("dns.server: port must be between 1 and 65535")
	}

	if len(s.AllowFrom) == 0 {
		add("dns.server: allow_from is required to define the client ACL")
	}

	for i, from := range s.AllowFrom {
		if !isValidIPOrCIDR(from) {
			add("dns.server.allow_from[%d]: %q is not a valid IP or CIDR", i, from)
		}
	}

	for i, ifName := range s.AllowInbound {
		if !isKnownInterface(ifName, cfg.Interfaces) {
			add("dns.server.allow_inbound[%d]: %q not found in interfaces", i, ifName)
		}
	}

	for i, up := range s.Upstreams {
		validateDNSUpstream(up, fmt.Sprintf("dns.server.upstreams[%d]", i), add)
	}

	if s.Threads < 0 {
		add("dns.server: threads must be positive")
	}

	zones := validateDNSZones(s.Zones, "dns.server", add)
	validateDNSForwards(s.Forward, "dns.server", zones, add)
	validateDNSCache(s.Cache, add)
	validateDNSViews(s, add)
}

func validateDNSViews(s *DNSServer, add addfunc) {
	if len(s.Views) == 0 {
		return
	}

	if len(s.Zones) > 0 {
		add("dns.server: views and top-level zones are mutually exclusive; " +
			"BIND requires every zone to live inside a view once any view exists")
	}

	if len(s.Forward) > 0 {
		add("dns.server: move forward entries into a view; " +
			"BIND requires every zone to live inside a view once any view exists")
	}

	names := map[string]string{}
	for i, v := range s.Views {
		path := fmt.Sprintf("dns.server.views[%d]", i)

		if v.Name == "" {
			add("%s: name is required", path)
		} else if prev, ok := names[v.Name]; ok {
			add("%s: duplicate view name %q, already declared at %s", path, v.Name, prev)
		} else {
			names[v.Name] = path
		}

		for j, from := range v.MatchFrom {
			if from == "any" || from == "none" || from == "localhost" || from == "localnets" {
				continue
			}

			if !isValidIPOrCIDR(from) {
				add("%s.match_from[%d]: %q is not a valid IP, CIDR or BIND acl keyword", path, j, from)
			}
		}

		for j, up := range v.Upstreams {
			validateDNSUpstream(up, fmt.Sprintf("%s.upstreams[%d]", path, j), add)
		}

		if len(v.Zones) == 0 && len(v.Forward) == 0 {
			add("%s: a view needs at least one zone or forward entry", path)
		}

		zones := validateDNSZones(v.Zones, path, add)
		validateDNSForwards(v.Forward, path, zones, add)
	}
}

func validateDNSUpstream(server, path string, add addfunc) {
	host := server
	if h, _, ok := strings.Cut(host, "#"); ok {
		host = h
	}

	host, port, hasPort := strings.Cut(host, "@")
	if !isValidIP(host) {
		add("%s: %q is not a valid IP address", path, server)
		return
	}

	if !hasPort {
		return
	}

	n, err := strconv.Atoi(port)
	if err != nil || !isValidPort(n) {
		add("%s: port %q must be between 1 and 65535", path, port)
	}
}

func validateDNSForwards(forwards []DNSForward, base string, zones map[string]string, add addfunc) {
	seen := map[string]string{}
	for i, f := range forwards {
		path := fmt.Sprintf("%s.forward[%d]", base, i)

		if f.Domain == "" {
			add("%s: domain is required", path)
			continue
		}

		if !isValidDNSName(f.Domain) {
			add("%s: domain %q is not a valid DNS name", path, f.Domain)
			continue
		}

		key := NormalizeDNSName(f.Domain)
		if prev, ok := seen[key]; ok {
			add("%s: duplicate forward domain %q, already declared at %s", path, f.Domain, prev)
		} else {
			seen[key] = path
		}

		if prev, ok := zones[key]; ok {
			add("%s: domain %q is also an authoritative zone at %s", path, f.Domain, prev)
		}

		if len(f.Servers) == 0 {
			add("%s: at least one server is required", path)
		}

		for j, srv := range f.Servers {
			validateDNSUpstream(srv, fmt.Sprintf("%s.servers[%d]", path, j), add)
		}
	}
}

func validateDNSCache(c *DNSCache, add addfunc) {
	if c == nil {
		return
	}

	if c.Size != "" && !isValidDNSCacheSize(c.Size) {
		add("dns.server.cache: size %q must be a byte count with an optional k, m, or g suffix", c.Size)
	}

	if c.RRSetSize != "" && !isValidDNSCacheSize(c.RRSetSize) {
		add("dns.server.cache: rrset_size %q must be a byte count with an optional k, m, or g suffix", c.RRSetSize)
	}

	if c.MinTTL < 0 || c.MaxTTL < 0 || c.MaxNegativeTTL < 0 {
		add("dns.server.cache: ttls must be positive")
	}

	if c.MinTTL > 0 && c.MaxTTL > 0 && c.MinTTL > c.MaxTTL {
		add("dns.server.cache: min_ttl %d must not exceed max_ttl %d", c.MinTTL, c.MaxTTL)
	}
}

func recordFQDN(zone, owner string) string {
	owner = strings.TrimSpace(owner)
	if owner == "@" || owner == "" {
		return NormalizeDNSName(zone)
	}

	if strings.HasSuffix(owner, ".") {
		return NormalizeDNSName(owner)
	}

	return NormalizeDNSName(owner + "." + zone)
}

func hasNameserverGlue(z DNSZone, ns string) bool {
	target := NormalizeDNSName(ns)
	if !strings.HasSuffix(target, "."+NormalizeDNSName(z.Name)) && target != NormalizeDNSName(z.Name) {
		return true
	}

	for _, r := range z.Records {
		switch strings.ToUpper(strings.TrimSpace(r.Type)) {
		case "A", "AAAA":
			if recordFQDN(z.Name, r.Name) == target {
				return true
			}
		}
	}

	return false
}

func validateDNSZones(zones []DNSZone, base string, add addfunc) map[string]string {
	names := map[string]string{}
	for i, z := range zones {
		path := fmt.Sprintf("%s.zones[%d]", base, i)

		if z.Name == "" {
			add("%s: name is required", path)
			continue
		}

		if !isValidDNSName(z.Name) {
			add("%s: name %q is not a valid DNS name", path, z.Name)
			continue
		}

		key := NormalizeDNSName(z.Name)
		if prev, ok := names[key]; ok {
			add("%s: duplicate zone name %q, already declared at %s", path, z.Name, prev)
		} else {
			names[key] = path
		}

		if z.TTL < 0 {
			add("%s: ttl must be positive", path)
		}

		switch {
		case len(z.Records) > 0 && len(z.Primaries) > 0:
			add("%s: records and primaries are mutually exclusive", path)
		case len(z.Records) == 0 && len(z.Primaries) == 0:
			add("%s: either records or primaries is required", path)
		}

		for j, ns := range z.Nameservers {
			if !isValidDNSName(ns) {
				add("%s.nameservers[%d]: %q is not a valid DNS name", path, j, ns)
				continue
			}

			if len(z.Records) > 0 && !hasNameserverGlue(z, ns) {
				add("%s.nameservers[%d]: %q is inside the zone but has no A or AAAA record; "+
					"add one or use a nameserver outside the zone", path, j, ns)
			}
		}

		for j, primary := range z.Primaries {
			if !isValidIP(primary) {
				add("%s.primaries[%d]: %q is not a valid IP address", path, j, primary)
			}
		}

		if len(z.Primaries) == 0 {
			if len(z.Nameservers) == 0 {
				add("%s: nameservers is required for a zone served from records", path)
			}

			if z.SOA == nil || z.SOA.Email == "" {
				add("%s: soa.email is required for a zone served from records", path)
			}
		}

		validateDNSSOA(z.SOA, path, add)
		validateDNSRecords(z, path, add)
	}

	return names
}

func validateDNSSOA(soa *DNSSOA, path string, add addfunc) {
	if soa == nil {
		return
	}

	if soa.Primary != "" && !isValidDNSName(soa.Primary) {
		add("%s.soa: primary %q is not a valid DNS name", path, soa.Primary)
	}

	if soa.Serial < 0 || soa.Refresh < 0 || soa.Retry < 0 || soa.Expire < 0 || soa.Minimum < 0 {
		add("%s.soa: serial and timers must be positive", path)
	}
}

var dnsRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT", "SRV", "PTR", "NS", "CAA", "SSHFP", "TLSA"}

func validateDNSRecords(z DNSZone, path string, add addfunc) {
	var owners []string
	byOwner := map[string][]string{}

	for i, r := range z.Records {
		p := fmt.Sprintf("%s.records[%d]", path, i)

		if r.Name == "" {
			add("%s: name is required", p)
		} else if !isValidZoneRecordName(r.Name, z.Name) {
			add("%s: name %q must be @, a name relative to %s, or an absolute name inside it", p, r.Name, z.Name)
		}

		recordType := strings.ToUpper(r.Type)
		if !slices.Contains(dnsRecordTypes, recordType) {
			add("%s: type %q must be one of %s", p, r.Type, strings.Join(dnsRecordTypes, ", "))
			continue
		}

		validateDNSRecordValue(recordType, r.Value, p, add)

		if r.Priority != 0 && recordType != "MX" && recordType != "SRV" {
			add("%s: priority is only supported on MX and SRV records", p)
		}

		if r.TTL < 0 {
			add("%s: ttl must be positive", p)
		}

		owner := strings.ToLower(r.Name)
		if _, ok := byOwner[owner]; !ok {
			owners = append(owners, owner)
		}

		byOwner[owner] = append(byOwner[owner], recordType)
	}

	for _, owner := range owners {
		types := byOwner[owner]
		cnameCount := 0
		for _, recordType := range types {
			if recordType == "CNAME" {
				cnameCount++
			}
		}

		if cnameCount > 1 {
			add("%s: owner %q has multiple CNAME records; only one CNAME target is allowed per owner", path, owner)
		}

		if cnameCount > 0 && len(types) > cnameCount {
			add("%s: owner %q has a CNAME alongside other records", path, owner)
		}
	}
}

func validateDNSRecordValue(recordType, value, path string, add addfunc) {
	if value == "" {
		add("%s: value is required", path)
		return
	}

	switch recordType {
	case "A":
		if net.ParseIP(value) == nil || isV6(value) {
			add("%s: A value %q is not an IPv4 address", path, value)
		}
	case "AAAA":
		if net.ParseIP(value) == nil || !isV6(value) {
			add("%s: AAAA value %q is not an IPv6 address", path, value)
		}
	case "CNAME", "NS", "PTR", "MX":
		if !isValidDNSName(value) {
			add("%s: %s value %q is not a valid DNS name", path, recordType, value)
		}
	case "SRV":
		validateSRVValue(value, path, add)
	}
}

func validateSRVValue(value, path string, add addfunc) {
	fields := strings.Fields(value)
	if len(fields) != 3 {
		add("%s: SRV value %q must be \"weight port target\"", path, value)
		return
	}

	if _, err := strconv.Atoi(fields[0]); err != nil {
		add("%s: SRV weight %q is not a number", path, fields[0])
	}

	port, err := strconv.Atoi(fields[1])
	if err != nil || !isValidPort(port) {
		add("%s: SRV port %q must be between 1 and 65535", path, fields[1])
	}

	if !isValidDNSName(fields[2]) {
		add("%s: SRV target %q is not a valid DNS name", path, fields[2])
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
	validateDDNS(cfg, add)
}

func validateDDNS(cfg *Config, add addfunc) {
	d := cfg.DHCP.DDNS
	if d == nil || !d.Enabled {
		return
	}

	if d.Domain == "" {
		add("dhcp.ddns.domain is required when dhcp.ddns.enabled is true")
	}

	if cfg.DNS == nil || cfg.DNS.Server == nil || !cfg.DNS.Server.Enabled {
		add("dhcp.ddns requires dns.server.enabled: true (the local BIND receives the updates)")
	}

	switch strings.ToLower(d.Algorithm) {
	case "", "hmac-md5", "hmac-sha1", "hmac-sha224", "hmac-sha256", "hmac-sha384", "hmac-sha512":
	default:
		add("dhcp.ddns.algorithm %q is not a supported TSIG algorithm", d.Algorithm)
	}

	switch d.ReplaceClientName {
	case "", "never", "always", "when-present", "when-not-present":
	default:
		add("dhcp.ddns.replace_client_name %q is invalid (never|always|when-present|when-not-present)", d.ReplaceClientName)
	}

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
			add("%s: interface %q not found in interfaces", p, s.Interface)
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
