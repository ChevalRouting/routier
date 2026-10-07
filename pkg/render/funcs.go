package render

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

func ifaceAddresses(cfg *config.Config, name string) []string {
	if i, ok := cfg.Interfaces[name]; ok {
		return i.Addresses
	}

	if t, ok := cfg.Tunnels[name]; ok {
		return t.Addresses
	}

	if w, ok := cfg.Wireguard[name]; ok {
		return w.Addresses
	}

	return nil
}

func neighborDefaultAF(addr string) string {
	ip := net.ParseIP(addr)
	if ip != nil && ip.To4() == nil {
		return "ipv6-unicast"
	}

	return "ipv4-unicast"
}

func isStaticAddr(addr string) bool {
	switch addr {
	case "dhcp", "dhcp4", "dhcp6", "slaac":
		return false
	}

	return true
}

func sanitizeNftName(s string) string {
	return strings.Map(sanitizeNftRune, s)
}

func sanitizeNftRune(r rune) rune {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
		return r
	}

	return '_'
}

func parseHostIP(s string) net.IP {
	s = strings.TrimSpace(s)
	if p := net.ParseIP(s); p != nil {
		return p
	}

	if h, _, err := net.ParseCIDR(s); err == nil {
		return h
	}

	return nil
}

func templateFuncs(data TemplateData) template.FuncMap {
	cfg := data.Config
	return template.FuncMap{
		"iface":      func(name string) string { return templateFuncsCallback(cfg, name) },
		"addr":       func(ifaceName string) string { return templateFuncsCallback2(cfg, ifaceName) },
		"addr4":      func(ifaceName string) string { return templateFuncsCallback3(cfg, ifaceName) },
		"addr6":      func(ifaceName string) string { return templateFuncsCallback4(cfg, ifaceName) },
		"network4":   func(ifaceName string) string { return templateFuncsCallback5(cfg, ifaceName) },
		"network6":   func(ifaceName string) string { return templateFuncsCallback6(cfg, ifaceName) },
		"gateway":    func(ifaceName string) string { return templateFuncsCallback7(cfg, ifaceName) },
		"gateway6":   func(ifaceName string) string { return templateFuncsCallback8(cfg, ifaceName) },
		"ip":         templateFuncsHandler,
		"subnet":     templateFuncsHandler2,
		"join":       strings.Join,
		"contains":   strings.Contains,
		"extraLines": templateFuncsHandler3,
		"quote": func(s string) string {
			return fmt.Sprintf("%q", s)
		},
		"default":    templateFuncsHandler4,
		"allDevices": func() []string { return templateFuncsCallback9(cfg) },
		"afName": func(s string) string {
			return strings.ReplaceAll(s, "-", " ")
		},
		"bgpAFs":            templateFuncsHandler5,
		"bgpNeighborInAF":   templateFuncsHandler6,
		"nftDefines":        func() string { return templateFuncsCallback10(&data, cfg) },
		"renderStr":         func(s string) (string, error) { return templateFuncsCallback11(&data, s) },
		"frrInterfaces":     func() []string { return templateFuncsCallback12(cfg) },
		"ospfIface":         func(name string) *config.OSPFInterface { return templateFuncsCallback13(cfg, name) },
		"ospf6Iface":        func(name string) *config.OSPF6Interface { return templateFuncsCallback14(cfg, name) },
		"pbrPolicy":         func(name string) string { return templateFuncsCallback15(cfg, name) },
		"bgpNeighborByDesc": func(desc string) string { return templateFuncsCallback16(cfg, desc) },
		"hasOSPF6":          func() bool { return templateFuncsCallback17(cfg) },
		"bgpNoRIB":          func() bool { return templateFuncsCallback18(cfg) },
		"bfdProfiles":       func() []config.BFDProfile { return templateFuncsCallback19(cfg) },
		"needBFD":           func() bool { return templateFuncsCallback20(cfg) },
		"nftRoutierTable": func() (string, error) {
			return renderRoutierTable(cfg)
		},
		"nftUserDefines": func() string { return templateFuncsCallback21(cfg) },
		"nftIncludes":    func() ([]string, error) { return templateFuncsCallback22(&data, cfg) },
		"dnsName": func(name string) string {
			return config.NormalizeDNSName(name)
		},
		"bindZoneFile": func(name string) string {
			return zoneFilePath(name)
		},
		"bindAllowFrom": func() []string {
			return bindAllowFrom(cfg)
		},
		"dnsNeedsNonlocalBind": func() bool { return templateFuncsCallback23(cfg) },
		"bindListen4":          func() string { return templateFuncsCallback24(cfg) },
		"bindListen6":          func() string { return templateFuncsCallback25(cfg) },
		"bindRecursion": func() bool {
			s := dnsServer(cfg)

			return s != nil && s.Recurses()
		},
		"bindViewRecursion": func(v *config.DNSView) bool {
			s := dnsServer(cfg)

			return v.Recurses(s != nil && s.Recurses())
		},
		"bindPort": func() int { return templateFuncsCallback26(cfg) },
		"bindValidateExcept": func() []string {
			return dnsInsecureDomains(cfg)
		},
		"bindDisableEmptyZones": func() []string {
			return dnsEmptyZoneOverrides(cfg)
		},
		"bindSize": func(size string) string {
			return bindCacheSize(size)
		},
		"ddnsManagedZone":     func(name string) bool { return ddnsManagedZone(cfg, name) },
		"ddnsUndeclaredZones": func() []string { return ddnsUndeclaredZones(cfg) },
		"ddnsEnabled":         func() bool { return ddnsActive(cfg) },
		"ddnsKeyName":         func() string { return DDNSKeyName },
		"ddnsKeyAlgorithm":    func() string { return bindTSIGAlgorithm(cfg) },
		"ddnsKeySecret":       func() string { return ddnsKeySecret(cfg) },
		"ddnsForwardZone":     func() string { return ddnsForwardZone(cfg) },
		"ddnsForwardZones":    func() []string { return ddnsForwardZones(cfg) },
		"ddnsReverseZones":    func() []string { return ddnsReverseZoneNames(cfg) },
		"bindZoneFileName":    func(name string) string { return zoneFilePath(name) },
		"bindControlAddr":     func() string { return "127.0.0.1" },
		"bindControlPort":     func() int { return 953 },
		"bindStatsFile":       func() string { return NamedStats },
		"bindLog":             func() string { return NamedLog },
	}
}

func emitVRRPDefines(b *strings.Builder, nftSet func([]string) string, emitAddr func(string, []string), prefix string, vips []string) {
	if len(vips) == 0 {
		return
	}

	emitAddr(prefix, vips)

	var hosts []string
	for _, addr := range vips {
		h, _, err := net.ParseCIDR(addr)
		if err != nil {
			if ip := net.ParseIP(strings.TrimSpace(addr)); ip != nil {
				hosts = append(hosts, ip.String())
			}

			continue
		}

		hosts = append(hosts, h.String())
	}

	if len(hosts) > 0 {
		_, _ = fmt.Fprintf(b, "define %s_addresses = %s\n", prefix, nftSet(hosts))
	}
}

type ownedChain struct {
	name      string
	header    string
	defPolicy string
	filter    bool
}

var ownedChains = []ownedChain{
	{name: "input", header: "type filter hook input priority filter", defPolicy: "drop", filter: true},
	{name: "forward", header: "type filter hook forward priority filter", defPolicy: "drop", filter: true},
	{name: "output", header: "type filter hook output priority filter", defPolicy: "accept", filter: true},
	{name: "prerouting", header: "type nat hook prerouting priority dstnat", defPolicy: "accept"},
	{name: "postrouting", header: "type nat hook postrouting priority srcnat", defPolicy: "accept"},
}

func renderRoutierTable(cfg *config.Config) (string, error) {
	managed := routierManagedRules(cfg)

	var b strings.Builder
	_, _ = b.WriteString("table inet routier {\n")

	for _, oc := range ownedChains {
		userLines, err := nftChainUserLines(cfg, oc.name)
		if err != nil {
			return "", err
		}

		if !oc.filter && !nftChainConfigured(cfg, oc.name) {
			continue
		}

		_, _ = fmt.Fprintf(&b, "\tchain %s {\n", oc.name)
		_, _ = fmt.Fprintf(&b, "\t\t%s; policy %s;\n", oc.header, nftChainPolicy(cfg, oc.name, oc.defPolicy))

		if auto := managed[oc.name]; len(auto) > 0 {
			_, _ = b.WriteString("\n\t\t# --- routier auto rules ---\n")
			for _, l := range auto {
				_, _ = fmt.Fprintf(&b, "\t\t%s\n", l)
			}
		}

		if len(userLines) > 0 {
			if len(managed[oc.name]) > 0 {
				_, _ = b.WriteString("\n\t\t# --- user rules ---\n")
			}

			for _, l := range userLines {
				if strings.TrimSpace(l) == "" {
					_, _ = b.WriteString("\n")
				} else {
					_, _ = fmt.Fprintf(&b, "\t\t%s\n", l)
				}
			}
		}

		_, _ = b.WriteString("\t}\n")
	}

	_, _ = b.WriteString("}\n")
	return b.String(), nil
}

func nftChainConfigured(cfg *config.Config, name string) bool {
	return cfg.Nftables != nil && cfg.Nftables.Chains[name] != nil
}

func nftChainPolicy(cfg *config.Config, name, def string) string {
	if cfg.Nftables != nil {
		if ch := cfg.Nftables.Chains[name]; ch != nil && ch.Policy != "" {
			return ch.Policy
		}
	}

	return def
}

func nftChainUserLines(cfg *config.Config, name string) ([]string, error) {
	if cfg.Nftables == nil {
		return nil, nil
	}

	ch := cfg.Nftables.Chains[name]
	if ch == nil {
		return nil, nil
	}

	lines := managedRuleLines(ch.Managed)

	if block := strings.TrimRight(ch.Rules, "\n"); block != "" {
		lines = append(lines, strings.Split(block, "\n")...)
	}

	for _, f := range ch.Files {
		p := f
		if !filepath.IsAbs(p) {
			p = filepath.Join(cfg.BaseDir, p)
		}

		content, rerr := os.ReadFile(p)
		if rerr != nil {
			log.Warn().Err(rerr).Str("file", p).Msg("nftables chain file: skipping unreadable file")
			continue
		}

		lines = append(lines, splitRuleLines(string(content))...)
	}

	return lines, nil
}

func routierManagedRules(cfg *config.Config) map[string][]string {
	out := map[string][]string{}
	add := func(chain, rule string) { out[chain] = append(out[chain], rule) }

	var wgPorts []int
	seenPort := map[int]bool{}
	for _, wg := range cfg.Wireguard {
		if wg.AllowInbound && wg.ListenPort > 0 && !seenPort[wg.ListenPort] {
			seenPort[wg.ListenPort] = true
			wgPorts = append(wgPorts, wg.ListenPort)
		}
	}

	if len(wgPorts) > 0 {
		sort.Ints(wgPorts)
		ports := make([]string, len(wgPorts))
		for i, p := range wgPorts {
			ports[i] = strconv.Itoa(p)
		}

		set := ports[0]
		if len(ports) > 1 {
			set = "{ " + strings.Join(ports, ", ") + " }"
		}

		add("input", fmt.Sprintf("udp dport %s accept comment \"routier: wireguard\"", set))
	}

	if cfg.HA != nil && cfg.HA.Conntrackd != nil {
		ct := cfg.HA.Conntrackd
		if ct.AllowInbound && ct.Interface != "" {
			port := ct.Port
			if port == 0 {
				port = 3780
			}

			add("input", fmt.Sprintf("iifname %q udp dport %d accept comment \"routier: conntrackd\"", resolveDevice(cfg, ct.Interface), port))
		}
	}

	if cfg.HA != nil {
		seenDev := map[string]bool{}
		for _, v := range cfg.HA.VRRP {
			if !v.AllowInbound {
				continue
			}

			target := v.Interface
			if v.Transport != "" {
				target = v.Transport
			}

			dev := resolveDevice(cfg, target)
			if dev == "" || seenDev[dev] {
				continue
			}

			seenDev[dev] = true
			add("input", fmt.Sprintf("iifname %q meta l4proto 112 accept comment \"routier: vrrp\"", dev))
		}
	}

	if r := cfg.Routing; r != nil {
		if r.BGP != nil {
			for _, ifn := range r.BGP.AllowInbound {
				out["input"] = append(out["input"], scopedAllow(cfg, ifn, "tcp dport 179", "bgp")...)
			}
		}

		if r.OSPF != nil {
			for _, ifn := range r.OSPF.AllowInbound {
				out["input"] = append(out["input"], scopedAllow(cfg, ifn, "meta l4proto 89", "ospf", "ip")...)
			}
		}

		if r.OSPF6 != nil {
			for _, ifn := range r.OSPF6.AllowInbound {
				out["input"] = append(out["input"], scopedAllow(cfg, ifn, "meta l4proto 89", "ospf6", "ip6")...)
			}
		}
	}

	if s := dnsServer(cfg); s != nil {
		for _, ifn := range s.AllowInbound {
			out["input"] = append(out["input"], dnsAllow(cfg, ifn)...)
		}
	}

	return out
}

func resolveDevice(cfg *config.Config, name string) string {
	if iface, ok := cfg.Interfaces[name]; ok && iface.Device != "" {
		return iface.Device
	}

	return name
}

func emitPrefixSet(b *strings.Builder, nftSet func([]string) string, prefix string, entries []string) {
	var v4, v6 []string
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}

		if strings.Contains(e, ":") {
			v6 = append(v6, e)
		} else {
			v4 = append(v4, e)
		}
	}

	if len(v4) > 0 {
		_, _ = fmt.Fprintf(b, "define %s = %s\n", prefix, nftSet(v4))
	}

	if len(v6) > 0 {
		_, _ = fmt.Fprintf(b, "define %s6 = %s\n", prefix, nftSet(v6))
	}
}

func dnsAllow(cfg *config.Config, ifaceName string) []string {
	s := dnsServer(cfg)
	if s == nil {
		return nil
	}

	if _, known := scopedAddresses(cfg, ifaceName); !known {
		log.Warn().Str("interface", ifaceName).
			Msg("dns allow_inbound: unknown interface, no firewall rule emitted")

		return nil
	}

	port := s.Port
	if port == 0 {
		port = 53
	}

	var v4, v6 bool
	for _, from := range s.AllowFrom {
		if strings.Contains(from, ":") {
			v6 = true
		} else {
			v4 = true
		}
	}

	san := sanitizeNftName(ifaceName)
	match := fmt.Sprintf("meta l4proto { tcp, udp } th dport %d accept comment %q", port, "routier: dns")

	var out []string
	if v4 {
		out = append(out, fmt.Sprintf("iifname $%s_interfaces ip saddr $dns_allow_from %s", san, match))
	}

	if v6 {
		out = append(out, fmt.Sprintf("iifname $%s_interfaces ip6 saddr $dns_allow_from6 %s", san, match))
	}

	return out
}

func addressFamilies(addrs []string) (v4, v6 bool) {
	for _, addr := range addrs {
		if !isStaticAddr(addr) {
			continue
		}

		h, _, err := net.ParseCIDR(addr)
		if err != nil {
			continue
		}

		if h.To4() != nil {
			v4 = true
		} else {
			v6 = true
		}
	}

	return
}

func scopedAddresses(cfg *config.Config, name string) ([]string, bool) {
	if iface, ok := cfg.Interfaces[name]; ok {
		return iface.Addresses, true
	}

	if wg, ok := cfg.Wireguard[name]; ok {
		return wg.Addresses, true
	}

	if t, ok := cfg.Tunnels[name]; ok {
		return t.Addresses, true
	}

	return nil, false
}

func scopedAllow(cfg *config.Config, ifaceName, match, comment string, fams ...string) []string {
	addrs, known := scopedAddresses(cfg, ifaceName)
	if !known {
		log.Warn().Str("interface", ifaceName).Str("feature", comment).
			Msg("allow_inbound: unknown interface, no firewall rule emitted")

		return nil
	}

	san := sanitizeNftName(ifaceName)
	v4, v6 := addressFamilies(addrs)

	want := func(f string) bool { return scopedAllowCallback(fams, f) }

	var out []string
	if v4 && want("ip") {
		out = append(out, fmt.Sprintf("iifname $%s_interfaces ip saddr $%s_network %s accept comment %q", san, san, match, "routier: "+comment))
	}

	if v6 && want("ip6") {
		out = append(out, fmt.Sprintf("iifname $%s_interfaces ip6 saddr $%s_network6 %s accept comment %q", san, san, match, "routier: "+comment))
	}

	if len(out) == 0 {
		out = append(out, fmt.Sprintf("iifname $%s_interfaces %s accept comment %q", san, match, "routier: "+comment))
	}

	return out
}

func splitRuleLines(s string) []string {
	var lines []string

	for _, raw := range strings.Split(s, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	return lines
}

func managedRuleLines(rules []config.ManagedRule) []string {
	var lines []string

	for _, rule := range rules {
		if rule.Disabled {
			continue
		}

		line := strings.Join(buildRuleParts(&rule), " ")

		comment := rule.Comment
		if comment == "" && rule.Tag != "" {
			comment = "routier:" + rule.Tag
		}

		if comment != "" {
			line = fmt.Sprintf("%s comment %q", line, comment)
		}

		lines = append(lines, line)
	}

	return lines
}

func buildRuleParts(rule *config.ManagedRule) []string {
	var parts []string

	m := rule.Match
	if m != nil {
		if m.CTState != "" {
			parts = append(parts, "ct state", m.CTState)
		}

		hasPort := m.SPort != "" || m.DPort != ""
		if m.Protocol != "" && !hasPort {
			parts = append(parts, "meta l4proto", m.Protocol)
		}

		if m.IIF != "" {
			parts = append(parts, "iifname", formatNftSet(m.IIF))
		}

		if m.OIF != "" {
			parts = append(parts, "oifname", formatNftSet(m.OIF))
		}

		af := m.AddrFamily
		if af == "" && (m.SAddr != "" || m.DAddr != "") {
			candidate := m.SAddr
			if candidate == "" {
				candidate = m.DAddr
			}

			if strings.Contains(candidate, ":") {
				af = "ip6"
			} else {
				af = "ip"
			}
		}

		if m.SAddr != "" {
			parts = append(parts, af+" saddr", formatNftSet(m.SAddr))
		}

		if m.DAddr != "" {
			parts = append(parts, af+" daddr", formatNftSet(m.DAddr))
		}

		if m.SPort != "" {
			proto := m.Protocol
			if proto == "" {
				proto = "tcp"
			}

			parts = append(parts, proto+" sport", formatNftSet(m.SPort))
		}

		if m.DPort != "" {
			proto := m.Protocol
			if proto == "" {
				proto = "tcp"
			}

			parts = append(parts, proto+" dport", formatNftSet(m.DPort))
		}
	}

	switch rule.Action {
	case "dnat", "snat":
		if rule.ActionTo == "" {
			parts = append(parts, rule.Action)
		} else if fam := natTargetFamily(rule.ActionTo); fam != "" {
			parts = append(parts, rule.Action, fam, "to", rule.ActionTo)
		} else {
			parts = append(parts, rule.Action+" to "+rule.ActionTo)
		}
	default:
		parts = append(parts, rule.Action)
	}

	return parts
}

func natTargetFamily(target string) string {
	host := target
	if strings.HasPrefix(host, "[") {
		if i := strings.Index(host, "]"); i > 0 {
			host = host[1:i]
		}
	} else if strings.Count(host, ":") == 1 {
		host = host[:strings.LastIndex(host, ":")]
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}

	if ip.To4() != nil {
		return "ip"
	}

	return "ip6"
}

func formatNftSet(v string) string {
	if strings.HasPrefix(v, "$") {
		return v
	}

	if strings.Contains(v, ",") {
		return "{ " + v + " }"
	}

	return v
}

type NftVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func NftVars(cfg *config.Config, opts ...Option) []NftVar {
	data := TemplateData{Config: cfg}
	for _, o := range opts {
		o(&data)
	}

	funcs := templateFuncs(data)
	definesFunc := funcs["nftDefines"].(func() string)
	raw := definesFunc()

	vars := []NftVar{}
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "define ") {
			continue
		}

		rest := strings.TrimPrefix(line, "define ")
		if idx := strings.Index(rest, " ="); idx > 0 {
			vars = append(vars, NftVar{
				Name:  rest[:idx],
				Value: strings.TrimSpace(rest[idx+2:]),
			})
		}
	}

	sort.Slice(vars, func(i, j int) bool { return vars[i].Name < vars[j].Name })

	return vars
}

func FriendNftVars(cfg *config.Config, name string, opts ...Option) []NftVar {
	prefix := "friends_" + sanitizeNftName(name) + "_"
	out := []NftVar{}

	for _, v := range NftVars(cfg, opts...) {
		if strings.HasPrefix(v.Name, prefix) {
			out = append(out, v)
		}
	}

	return out
}

func templateFuncsHandler(cidr string) string {
	h, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr
	}

	return h.String()
}

func templateFuncsHandler2(cidr string) string {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr
	}

	return n.String()
}

func templateFuncsHandler3(lines []string, indent string) string {
	var b strings.Builder
	for _, l := range lines {
		if l = strings.TrimSpace(l); l != "" {
			_, _ = b.WriteString("\n" + indent + l)
		}
	}

	return b.String()
}

func templateFuncsHandler4(def, val string) string {
	if val == "" {
		return def
	}

	return val
}

func templateFuncsHandler5(bgp *config.BGP) []string {
	seen := map[string]struct{}{}
	for af := range bgp.AddressFamilies {
		seen[af] = struct{}{}
	}

	for _, n := range bgp.Neighbors {
		if len(n.AddressFamilies) > 0 {
			for af := range n.AddressFamilies {
				seen[af] = struct{}{}
			}
		} else {
			seen[neighborDefaultAF(n.Address)] = struct{}{}
		}
	}

	afs := make([]string, 0, len(seen))
	for af := range seen {
		afs = append(afs, af)
	}

	sort.Strings(afs)
	return afs
}

func templateFuncsHandler6(n config.BGPNeighbor, af string) *config.BGPNeighborAF {
	if len(n.AddressFamilies) > 0 {
		return n.AddressFamilies[af]
	}

	if neighborDefaultAF(n.Address) == af {
		return &config.BGPNeighborAF{}
	}

	return nil
}

func templateFuncsHandler7(vals []string) string {
	if len(vals) == 1 {
		return vals[0]
	}

	return "{ " + strings.Join(vals, ", ") + " }"
}

func templateFuncsHandler8(bgp *config.BGP) bool {
	if bgp == nil {
		return false
	}

	for _, n := range bgp.Neighbors {
		if n.BFD {
			return true
		}
	}

	return false
}

func templateFuncsCallback(cfg *config.Config, name string) string {
	if i, ok := cfg.Interfaces[name]; ok {
		return i.Device
	}

	return name
}

func templateFuncsCallback2(cfg *config.Config, ifaceName string) string {
	addrs := ifaceAddresses(cfg, ifaceName)
	if len(addrs) > 0 {
		return addrs[0]
	}

	return ""
}

func templateFuncsCallback3(cfg *config.Config, ifaceName string) string {
	addrs := ifaceAddresses(cfg, ifaceName)
	for _, a := range addrs {
		h, _, err := net.ParseCIDR(a)
		if err != nil || h.To4() == nil {
			continue
		}

		return h.String()
	}

	return ""
}

func templateFuncsCallback4(cfg *config.Config, ifaceName string) string {
	addrs := ifaceAddresses(cfg, ifaceName)
	for _, a := range addrs {
		h, _, err := net.ParseCIDR(a)
		if err != nil || h.To4() != nil {
			continue
		}

		return h.String()
	}

	return ""
}

func templateFuncsCallback5(cfg *config.Config, ifaceName string) string {
	addrs := ifaceAddresses(cfg, ifaceName)
	for _, a := range addrs {
		h, n, err := net.ParseCIDR(a)
		if err != nil || h.To4() == nil {
			continue
		}

		return n.String()
	}

	return ""
}

func templateFuncsCallback6(cfg *config.Config, ifaceName string) string {
	addrs := ifaceAddresses(cfg, ifaceName)
	for _, a := range addrs {
		h, n, err := net.ParseCIDR(a)
		if err != nil || h.To4() != nil {
			continue
		}

		return n.String()
	}

	return ""
}

func templateFuncsCallback7(cfg *config.Config, ifaceName string) string {
	if cfg.Routing == nil {
		return ""
	}

	dev := ""
	if i, ok := cfg.Interfaces[ifaceName]; ok {
		dev = i.Device
	}

	for _, r := range cfg.Routing.Static {
		if r.Destination != "0.0.0.0/0" || r.Via == "" {
			continue
		}

		if r.Dev != "" && r.Dev != dev && r.Dev != ifaceName {
			continue
		}

		if !strings.Contains(r.Via, ":") {
			return r.Via
		}
	}

	return ""
}

func templateFuncsCallback8(cfg *config.Config, ifaceName string) string {
	if cfg.Routing == nil {
		return ""
	}

	dev := ""
	if i, ok := cfg.Interfaces[ifaceName]; ok {
		dev = i.Device
	}

	for _, r := range cfg.Routing.Static {
		if r.Destination != "::/0" || r.Via == "" {
			continue
		}

		if r.Dev != "" && r.Dev != dev && r.Dev != ifaceName {
			continue
		}

		if strings.Contains(r.Via, ":") {
			return r.Via
		}
	}

	return ""
}

func templateFuncsCallback9(cfg *config.Config) []string {
	var devs []string
	for _, i := range cfg.Interfaces {
		devs = append(devs, i.Device)
	}

	return devs
}

func templateFuncsCallback10(data *TemplateData, cfg *config.Config) string {
	var b strings.Builder

	nftSet := templateFuncsHandler7

	var collectAddrs func(addresses []string)

	emitAddrDefines := func(prefix string, addresses []string) {
		templateFuncsCallback10Callback(&b, nftSet, prefix, addresses)
	}

	writeAddrDefines := func(prefix string, addresses []string) {
		collectAddrs(addresses)
		emitAddrDefines(prefix, addresses)
	}

	var ifaceDevs, tunnelDevs, wgDevs []string
	var meV4, meV6 []string
	vrfMembers := map[string][]string{}

	collectAddrs = func(addresses []string) { templateFuncsCallback10Callback2(&meV4, &meV6, addresses) }

	ifaceNames := make([]string, 0, len(cfg.Interfaces))
	for n := range cfg.Interfaces {
		ifaceNames = append(ifaceNames, n)
	}

	sort.Strings(ifaceNames)

	vrrpByIface := map[string][]string{}
	vrrpByID := map[int][]string{}
	var vrrpIDs []int

	if cfg.HA != nil {
		for _, v := range cfg.HA.VRRP {
			vrrpByIface[v.Interface] = append(vrrpByIface[v.Interface], v.VIPs...)
			if _, seen := vrrpByID[v.ID]; !seen {
				vrrpIDs = append(vrrpIDs, v.ID)
			}

			vrrpByID[v.ID] = append(vrrpByID[v.ID], v.VIPs...)
		}
	}

	for _, name := range ifaceNames {
		iface := cfg.Interfaces[name]
		sanitized := sanitizeNftName(name)
		dev := iface.Device
		if dev == "" {
			dev = iface.Select
		}

		_, _ = fmt.Fprintf(&b, "define %s_interfaces = \"%s\"\n", sanitized, dev)
		if dev != "" {
			ifaceDevs = append(ifaceDevs, "\""+dev+"\"")
			if iface.VRF != "" {
				vrfMembers[iface.VRF] = append(vrfMembers[iface.VRF], "\""+dev+"\"")
			}
		}

		writeAddrDefines(sanitized, iface.Addresses)

		if vips := vrrpByIface[name]; len(vips) > 0 {
			emitVRRPDefines(&b, nftSet, emitAddrDefines, sanitized+"_vrrp", vips)
		}

		if cfg.Routing != nil {
			for _, r := range cfg.Routing.Static {
				if r.Via == "" || r.Dev != iface.Device {
					continue
				}

				if r.Destination != "0.0.0.0/0" && r.Destination != "::/0" {
					continue
				}

				if strings.Contains(r.Via, ":") {
					_, _ = fmt.Fprintf(&b, "define %s_gateway6 = %s\n", sanitized, r.Via)
				} else {
					_, _ = fmt.Fprintf(&b, "define %s_gateway = %s\n", sanitized, r.Via)
				}
			}
		}
	}

	sort.Ints(vrrpIDs)
	for _, id := range vrrpIDs {
		emitVRRPDefines(&b, nftSet, emitAddrDefines, "vrrp_"+strconv.Itoa(id), vrrpByID[id])
	}

	tunnelNames := make([]string, 0, len(cfg.Tunnels))
	for n := range cfg.Tunnels {
		tunnelNames = append(tunnelNames, n)
	}

	sort.Strings(tunnelNames)

	for _, name := range tunnelNames {
		tunnel := cfg.Tunnels[name]
		sanitized := sanitizeNftName(name)
		_, _ = fmt.Fprintf(&b, "define %s_interfaces = \"%s\"\n", sanitized, name)
		tunnelDevs = append(tunnelDevs, "\""+name+"\"")
		writeAddrDefines(sanitized, tunnel.Addresses)
	}

	wgNames := make([]string, 0, len(cfg.Wireguard))
	for n := range cfg.Wireguard {
		wgNames = append(wgNames, n)
	}

	sort.Strings(wgNames)

	for _, name := range wgNames {
		wg := cfg.Wireguard[name]
		sanitized := sanitizeNftName(name)
		_, _ = fmt.Fprintf(&b, "define %s_interfaces = \"%s\"\n", sanitized, name)
		wgDevs = append(wgDevs, "\""+name+"\"")
		writeAddrDefines(sanitized, wg.Addresses)
	}

	vrfNames := make([]string, 0, len(cfg.VRFs))
	for n := range cfg.VRFs {
		vrfNames = append(vrfNames, n)
	}

	sort.Strings(vrfNames)

	var vrfDevs []string
	for _, name := range vrfNames {
		sanitized := sanitizeNftName(name)
		_, _ = fmt.Fprintf(&b, "define vrf_%s_interfaces = \"%s\"\n", sanitized, name)
		if members := vrfMembers[name]; len(members) > 0 {
			_, _ = fmt.Fprintf(&b, "define vrf_%s_members = %s\n", sanitized, nftSet(members))
		}

		vrfDevs = append(vrfDevs, "\""+name+"\"")
	}

	if len(ifaceDevs) > 0 {
		_, _ = fmt.Fprintf(&b, "define interfaces = %s\n", nftSet(ifaceDevs))
	}

	if len(tunnelDevs) > 0 {
		_, _ = fmt.Fprintf(&b, "define tunnels = %s\n", nftSet(tunnelDevs))
	}

	if len(wgDevs) > 0 {
		_, _ = fmt.Fprintf(&b, "define wireguard = %s\n", nftSet(wgDevs))
	}

	if len(vrfDevs) > 0 {
		_, _ = fmt.Fprintf(&b, "define vrfs = %s\n", nftSet(vrfDevs))
	}

	if cfg.DNS != nil && len(cfg.DNS.Nameservers) > 0 {
		_, _ = fmt.Fprintf(&b, "define dns_nameservers = %s\n", nftSet(cfg.DNS.Nameservers))
	}

	if s := dnsServer(cfg); s != nil {
		emitPrefixSet(&b, nftSet, "dns_allow_from", s.AllowFrom)
		emitPrefixSet(&b, nftSet, "dns_listen", dnsListenAddresses(cfg))
	}

	if cfg.Routing != nil && cfg.Routing.Anycast != nil {
		for _, svc := range cfg.Routing.Anycast.Services {
			for _, ip := range svc.AnycastIPs {
				p := parseHostIP(ip)
				if p == nil {
					continue
				}

				if p.To4() != nil {
					meV4 = append(meV4, p.String())
				} else {
					meV6 = append(meV6, p.String())
				}
			}
		}
	}

	if len(meV4) > 0 {
		_, _ = fmt.Fprintf(&b, "define me = %s\n", nftSet(meV4))
	}

	if len(meV6) > 0 {
		_, _ = fmt.Fprintf(&b, "define me6 = %s\n", nftSet(meV6))
	}

	emitIPSet := func(prefix string, ips []string) { templateFuncsCallback10Callback3(&b, nftSet, prefix, ips) }

	if cfg.Routing != nil {
		rt := cfg.Routing
		if rt.BGP != nil {
			neighbors := make([]string, 0, len(rt.BGP.Neighbors))
			for _, n := range rt.BGP.Neighbors {
				neighbors = append(neighbors, n.Address)
				emitIPSet("bgp_neighbor_"+sanitizeNftName(n.Address), []string{n.Address})

				if n.Description != "" {
					emitIPSet("bgp_neighbor_"+sanitizeNftName(n.Description), []string{n.Address})
				}
			}

			emitIPSet("bgp_neighbors", neighbors)
		}

		if rt.OSPF != nil && len(rt.OSPF.Interfaces) > 0 {
			ospfIfs := make([]string, 0, len(rt.OSPF.Interfaces))

			for name := range rt.OSPF.Interfaces {
				ospfIfs = append(ospfIfs, name)
			}

			sort.Strings(ospfIfs)
			_, _ = fmt.Fprintf(&b, "define ospf_interfaces = %s\n", nftSet(ospfIfs))
		}

		if rt.Anycast != nil {
			var allIPs, allEndpoints []string
			for _, svc := range rt.Anycast.Services {
				san := sanitizeNftName(svc.Name)
				emitIPSet("anycast_"+san, svc.AnycastIPs)

				endpoints := make([]string, 0, len(svc.Endpoints))
				for _, e := range svc.Endpoints {
					endpoints = append(endpoints, e.IP)
				}

				emitIPSet("anycast_"+san+"_endpoints", endpoints)
				allIPs = append(allIPs, svc.AnycastIPs...)
				allEndpoints = append(allEndpoints, endpoints...)
			}

			emitIPSet("anycast_ips", allIPs)
			emitIPSet("anycast_endpoints", allEndpoints)
		}
	}

	friendNames := make([]string, 0, len((*data).Friends))
	for n := range (*data).Friends {
		friendNames = append(friendNames, n)
	}

	sort.Strings(friendNames)
	for _, fname := range friendNames {
		fv := (*data).Friends[fname]
		prefix := "friends_" + sanitizeNftName(fname)

		if len(fv.Exports) > 0 {
			exportNames := make([]string, 0, len(fv.Exports))
			for n := range fv.Exports {
				exportNames = append(exportNames, n)
			}

			sort.Strings(exportNames)
			for _, en := range exportNames {
				_, _ = fmt.Fprintf(&b, "define %s_%s = %s\n", prefix, en, fv.Exports[en])
			}

			continue
		}

		ifaceNames := make([]string, 0, len(fv.Interfaces))
		for n := range fv.Interfaces {
			ifaceNames = append(ifaceNames, n)
		}

		sort.Strings(ifaceNames)
		for _, iname := range ifaceNames {
			emitAddrDefines(prefix+"_"+sanitizeNftName(iname), fv.Interfaces[iname])
		}
	}

	return b.String()
}

func templateFuncsCallback11(data *TemplateData, s string) (string, error) {
	funcs := templateFuncs((*data))
	t, err := template.New("render-str").Funcs(funcs).Parse(s)
	if err != nil {
		return "", fmt.Errorf("renderStr: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, (*data)); err != nil {
		return "", fmt.Errorf("renderStr: %w", err)
	}

	return buf.String(), nil
}

func templateFuncsCallback12(cfg *config.Config) []string {
	seen := map[string]struct{}{}
	if cfg.Routing != nil {
		if cfg.Routing.OSPF != nil {
			for name := range cfg.Routing.OSPF.Interfaces {
				seen[name] = struct{}{}
			}
		}

		if cfg.Routing.OSPF6 != nil {
			for name := range cfg.Routing.OSPF6.Interfaces {
				seen[name] = struct{}{}
			}
		}

		if cfg.Routing.PBR != nil {
			for name := range cfg.Routing.PBR.Policies {
				seen[name] = struct{}{}
			}
		}

		for name := range cfg.Interfaces {
			seen[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		if name != "" {
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names
}

func templateFuncsCallback13(cfg *config.Config, name string) *config.OSPFInterface {
	if cfg.Routing == nil || cfg.Routing.OSPF == nil {
		return nil
	}

	return cfg.Routing.OSPF.Interfaces[name]
}

func templateFuncsCallback14(cfg *config.Config, name string) *config.OSPF6Interface {
	if cfg.Routing == nil || cfg.Routing.OSPF6 == nil {
		return nil
	}

	return cfg.Routing.OSPF6.Interfaces[name]
}

func templateFuncsCallback15(cfg *config.Config, name string) string {
	if cfg.Routing == nil || cfg.Routing.PBR == nil {
		return ""
	}

	return cfg.Routing.PBR.Policies[name]
}

func templateFuncsCallback16(cfg *config.Config, desc string) string {
	if cfg.Routing == nil {
		return ""
	}

	if cfg.Routing.BGP != nil {
		for _, n := range cfg.Routing.BGP.Neighbors {
			if n.Description == desc {
				return n.Address
			}
		}
	}

	for _, vrf := range cfg.Routing.VRFs {
		if vrf.BGP == nil {
			continue
		}

		for _, n := range vrf.BGP.Neighbors {
			if n.Description == desc {
				return n.Address
			}
		}
	}

	return ""
}

func templateFuncsCallback17(cfg *config.Config) bool {
	if cfg.Routing == nil {
		return false
	}

	if cfg.Routing.OSPF6 != nil {
		return true
	}

	for _, vr := range cfg.Routing.VRFs {
		if vr.OSPF6 != nil {
			return true
		}
	}

	return false
}

func templateFuncsCallback18(cfg *config.Config) bool {
	if cfg.Routing == nil {
		return false
	}

	if cfg.Routing.BGP != nil && cfg.Routing.BGP.NoRIB {
		return true
	}

	for _, vr := range cfg.Routing.VRFs {
		if vr.BGP != nil && vr.BGP.NoRIB {
			return true
		}
	}

	return false
}

func templateFuncsCallback19(cfg *config.Config) []config.BFDProfile {
	if cfg.Routing == nil || cfg.Routing.BFD == nil {
		return nil
	}

	return cfg.Routing.BFD.Profiles
}

func templateFuncsCallback20(cfg *config.Config) bool {
	if cfg.Routing == nil {
		return false
	}

	if cfg.Routing.BFD != nil && len(cfg.Routing.BFD.Profiles) > 0 {
		return true
	}

	bgpHasBFD := templateFuncsHandler8

	if bgpHasBFD(cfg.Routing.BGP) {
		return true
	}

	for _, vr := range cfg.Routing.VRFs {
		if bgpHasBFD(vr.BGP) {
			return true
		}
	}

	return false
}

func templateFuncsCallback21(cfg *config.Config) string {
	if cfg.Nftables == nil {
		return ""
	}

	return strings.TrimRight(cfg.Nftables.Defines, "\n")
}

func templateFuncsCallback22(data *TemplateData, cfg *config.Config) ([]string, error) {
	if cfg.Nftables == nil {
		return nil, nil
	}

	funcs := templateFuncs((*data))
	var out []string
	for _, p := range cfg.Nftables.Include {
		path := p
		if !filepath.IsAbs(path) {
			path = filepath.Join(cfg.BaseDir, path)
		}

		content, rerr := os.ReadFile(path)
		if rerr != nil {
			log.Warn().Err(rerr).Str("file", path).Msg("nftables include: skipping unreadable file")
			continue
		}

		t, terr := template.New("nft-include").Funcs(funcs).Parse(string(content))
		if terr != nil {
			return nil, fmt.Errorf("nftables include %s: %w", path, terr)
		}

		var buf bytes.Buffer
		if terr := t.Execute(&buf, (*data)); terr != nil {
			return nil, fmt.Errorf("nftables include %s: %w", path, terr)
		}

		out = append(out, buf.String())
	}

	return out, nil
}

func templateFuncsCallback23(cfg *config.Config) bool {
	if _, ok := cfg.Sysctl["net.ipv4.ip_nonlocal_bind"]; ok {
		return false
	}

	return DNSListensOnVIP(cfg)
}

func templateFuncsCallback24(cfg *config.Config) string {
	if dnsListenAny(cfg) {
		return "any;"
	}

	v4, _ := splitListenFamilies(dnsListenAddresses(cfg))
	if len(v4) == 0 {
		return "127.0.0.1;"
	}

	return bindAddressList(v4)
}

func templateFuncsCallback25(cfg *config.Config) string {
	if dnsListenAny(cfg) {
		return "any;"
	}

	_, v6 := splitListenFamilies(dnsListenAddresses(cfg))

	return bindAddressList(v6)
}

func templateFuncsCallback26(cfg *config.Config) int {
	if s := dnsServer(cfg); s != nil && s.Port > 0 {
		return s.Port
	}

	return 53
}

func scopedAllowCallback(fams []string, f string) bool {
	if len(fams) == 0 {
		return true
	}

	return slices.Contains(fams, f)
}

func templateFuncsCallback10Callback(b *strings.Builder, nftSet func(vals []string) string, prefix string, addresses []string) {
	var v4h, v4n, v6h, v6n []string
	for _, addr := range addresses {
		if !isStaticAddr(addr) {
			continue
		}

		h, n, err := net.ParseCIDR(addr)
		if err != nil {
			ip := net.ParseIP(strings.TrimSpace(addr))
			if ip == nil {
				continue
			}

			if ip.To4() != nil {
				v4h = append(v4h, ip.String())
			} else {
				v6h = append(v6h, ip.String())
			}

			continue
		}

		if h.To4() != nil {
			v4h = append(v4h, h.String())
			v4n = append(v4n, n.String())
		} else {
			v6h = append(v6h, h.String())
			v6n = append(v6n, n.String())
		}
	}

	if len(v4h) > 0 {
		_, _ = fmt.Fprintf(b, "define %s_address = %s\n", prefix, nftSet(v4h))
	}

	if len(v4n) > 0 {
		_, _ = fmt.Fprintf(b, "define %s_network = %s\n", prefix, nftSet(v4n))
	}

	if len(v6h) > 0 {
		_, _ = fmt.Fprintf(b, "define %s_address6 = %s\n", prefix, nftSet(v6h))
	}

	if len(v6n) > 0 {
		_, _ = fmt.Fprintf(b, "define %s_network6 = %s\n", prefix, nftSet(v6n))
	}
}

func templateFuncsCallback10Callback2(meV4 *[]string, meV6 *[]string, addresses []string) {
	for _, addr := range addresses {
		if !isStaticAddr(addr) {
			continue
		}

		h, _, err := net.ParseCIDR(addr)
		if err != nil {
			continue
		}

		if h.To4() != nil {
			(*meV4) = append((*meV4), h.String())
		} else {
			(*meV6) = append((*meV6), h.String())
		}
	}
}

func templateFuncsCallback10Callback3(b *strings.Builder, nftSet func(vals []string) string, prefix string, ips []string) {
	var v4, v6 []string
	for _, ip := range ips {
		p := parseHostIP(ip)
		if p == nil {
			continue
		}

		if p.To4() != nil {
			v4 = append(v4, p.String())
		} else {
			v6 = append(v6, p.String())
		}
	}

	if len(v4) > 0 {
		_, _ = fmt.Fprintf(b, "define %s = %s\n", prefix, nftSet(v4))
	}

	if len(v6) > 0 {
		_, _ = fmt.Fprintf(b, "define %s6 = %s\n", prefix, nftSet(v6))
	}
}
