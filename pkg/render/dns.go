package render

import hash "hash"

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	NamedLog      = "/var/log/named/named.log"
	NamedZoneDir  = "/etc/bind/zones"
	NamedConfDest = "/etc/bind/named.conf"
	NamedStats    = "/var/bind/named.stats"
	NamedRunDir   = "/run/named"

	NamedZoneName = "bind/zones/"
	NamedConfName = "bind/named.conf"

	NamedStaticDir = "/var/lib/routier/dns-static"

	namedZonePfx    = NamedZoneName
	namedZoneDest   = NamedZoneDir + "/"
	namedStaticDest = NamedStaticDir + "/"
	namedZoneSufx   = "zone"
)

func ZoneOriginFromName(name string) string {
	base := strings.TrimPrefix(name, namedZonePfx)

	return strings.TrimSuffix(strings.TrimSuffix(base, namedZoneSufx), ".")
}

func NamedZonesFromNames(names []string) []string {
	var zones []string
	for _, n := range names {
		if strings.HasPrefix(n, namedZonePfx) {
			zones = append(zones, ZoneOriginFromName(n))
		}
	}

	return zones
}

const (
	defaultZoneTTL    = 3600
	defaultSOARefresh = 3600
	defaultSOARetry   = 600
	defaultSOAExpire  = 604800
	defaultSOAMinimum = 300
)

func dnsServer(cfg *config.Config) *config.DNSServer {
	if cfg == nil || cfg.DNS == nil || cfg.DNS.Server == nil || !cfg.DNS.Server.Enabled {
		return nil
	}

	return cfg.DNS.Server
}

func LocalDNSServerEnabled(cfg *config.Config) bool {
	return dnsServer(cfg) != nil
}

func ZoneFileName(name string) string {
	return config.NormalizeDNSName(name) + namedZoneSufx
}

func zoneFilePath(name string) string {
	return namedZoneDest + ZoneFileName(name)
}

func staticShadowPath(name string) string {
	return namedStaticDest + ZoneFileName(name)
}

func DNSListenAddresses(cfg *config.Config) []string {
	return dnsListenAddresses(cfg)
}

func dnsListenAddresses(cfg *config.Config) []string {
	s := dnsServer(cfg)
	if s == nil {
		return nil
	}

	seen := make(map[string]bool)
	var out []string
	for _, entry := range s.Listen {
		addrs, err := config.ResolveListen(cfg, entry)
		if err != nil {
			log.Warn().Err(err).Str("listen", entry).Msg("dns: skipping unresolvable listen address")
			continue
		}

		for _, addr := range addrs {
			if !seen[addr] {
				seen[addr] = true
				out = append(out, addr)
			}
		}
	}

	sortIPStrings(out)
	return out
}

func DNSListensOnVIP(cfg *config.Config) bool {
	s := dnsServer(cfg)
	if s == nil {
		return false
	}

	for _, entry := range s.Listen {
		if ref, ok := config.ParseListenRef(entry); ok && ref.Kind == config.ListenVIPs {
			return true
		}
	}

	return false
}

func sortIPStrings(addrs []string) {
	sort.Slice(addrs, func(i, j int) bool { return sortIPStringsCallback(addrs, i, j) })
}

func bindAllowFrom(cfg *config.Config) []string {
	s := dnsServer(cfg)
	if s == nil {
		return nil
	}

	out := make([]string, 0, len(s.AllowFrom))
	seen := make(map[string]bool)
	for _, from := range s.AllowFrom {
		from = strings.TrimSpace(from)
		if from == "" || seen[from] {
			continue
		}

		seen[from] = true
		out = append(out, from)
	}

	return out
}

func splitListenFamilies(addrs []string) (v4, v6 []string) {
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}

		if ip.To4() != nil {
			v4 = append(v4, addr)
		} else {
			v6 = append(v6, addr)
		}
	}

	return
}

func dnsListenAny(cfg *config.Config) bool {
	for _, addr := range dnsListenAddresses(cfg) {
		if ip := net.ParseIP(addr); ip != nil && ip.IsUnspecified() {
			return true
		}
	}

	return false
}

func DNSQueryAddress(cfg *config.Config) string {
	addrs := dnsListenAddresses(cfg)
	if len(addrs) == 0 {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.IsUnspecified() {
			if ip.To4() != nil {
				return "127.0.0.1"
			}

			return "::1"
		}
	}

	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.IsLoopback() {
			return addr
		}
	}

	return addrs[0]
}

func bindAddressList(addrs []string) string {
	if len(addrs) == 0 {
		return ""
	}

	var b strings.Builder
	for _, addr := range addrs {
		_, _ = b.WriteString(addr)
		_, _ = b.WriteString("; ")
	}

	return strings.TrimSpace(b.String())
}

func dnsInsecureDomains(cfg *config.Config) []string {
	s := dnsServer(cfg)
	if s == nil {
		return nil
	}

	var names []string
	seen := make(map[string]bool)
	collect := func(raw string, signed bool) { dnsInsecureDomainsCallback(&names, seen, raw, signed) }

	for _, f := range s.Forward {
		collect(f.Domain, f.DNSSEC)
	}

	for _, z := range s.Zones {
		collect(z.Name, z.DNSSEC)
	}

	for _, v := range s.Views {
		for _, f := range v.Forward {
			collect(f.Domain, f.DNSSEC)
		}

		for _, z := range v.Zones {
			collect(z.Name, z.DNSSEC)
		}
	}

	return names
}

func dnsEmptyZoneOverrides(cfg *config.Config) []string {
	var out []string
	for _, name := range dnsInsecureDomains(cfg) {
		if bindHasEmptyZone(name) {
			out = append(out, name)
		}
	}

	return out
}

var bindEmptyZoneSuffixes = []string{
	"in-addr.arpa",
	"ip6.arpa",
	"home.arpa",
	"resolver.arpa",
	"service.arpa",
	"empty.arpa",
}

func bindHasEmptyZone(name string) bool {
	lower := strings.ToLower(strings.TrimSuffix(name, "."))
	for _, suffix := range bindEmptyZoneSuffixes {
		if lower == suffix || strings.HasSuffix(lower, "."+suffix) {
			return true
		}
	}

	return false
}

func bindCacheSize(size string) string {
	trimmed := strings.TrimSpace(size)
	if trimmed == "" {
		return ""
	}

	last := trimmed[len(trimmed)-1]
	if last >= '0' && last <= '9' {
		return trimmed
	}

	return trimmed[:len(trimmed)-1] + strings.ToUpper(string(last))
}

type zoneLine struct {
	owner string
	ttl   string
	rtype string
	data  string
}

type soaValues struct {
	Primary string
	Email   string
	Serial  int
	Refresh int
	Retry   int
	Expire  int
	Minimum int
}

func soaEmail(email string) string {
	local, domain, ok := strings.Cut(strings.TrimSpace(email), "@")
	if !ok {
		return config.NormalizeDNSName(email)
	}

	return config.NormalizeDNSName(strings.ReplaceAll(local, ".", `\.`) + "." + domain)
}

func zoneSOA(z config.DNSZone) soaValues {
	v := soaValues{
		Refresh: defaultSOARefresh,
		Retry:   defaultSOARetry,
		Expire:  defaultSOAExpire,
		Minimum: defaultSOAMinimum,
	}
	if len(z.Nameservers) > 0 {
		v.Primary = config.NormalizeDNSName(z.Nameservers[0])
	}

	if soa := z.SOA; soa != nil {
		if soa.Primary != "" {
			v.Primary = config.NormalizeDNSName(soa.Primary)
		}

		v.Email = soaEmail(soa.Email)
		v.Serial = soa.Serial

		if soa.Refresh > 0 {
			v.Refresh = soa.Refresh
		}

		if soa.Retry > 0 {
			v.Retry = soa.Retry
		}

		if soa.Expire > 0 {
			v.Expire = soa.Expire
		}

		if soa.Minimum > 0 {
			v.Minimum = soa.Minimum
		}
	}

	if v.Primary == "" {
		v.Primary = config.NormalizeDNSName("ns." + z.Name)
	}

	if v.Email == "" {
		v.Email = config.NormalizeDNSName("hostmaster." + z.Name)
	}

	return v
}

func zoneTTL(z config.DNSZone) int {
	if z.TTL > 0 {
		return z.TTL
	}

	return defaultZoneTTL
}

func recordTTL(ttl int) string {
	if ttl <= 0 {
		return ""
	}

	return strconv.Itoa(ttl)
}

func quoteTXT(value string) string {
	if len(value) > 1 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value
	}

	return fmt.Sprintf("%q", value)
}

func recordData(r config.DNSRecord) string {
	value := strings.TrimSpace(r.Value)
	switch strings.ToUpper(r.Type) {
	case "TXT":
		return quoteTXT(value)
	case "MX", "SRV":
		return strconv.Itoa(r.Priority) + " " + value
	}

	return value
}

func zoneLines(z config.DNSZone) []zoneLine {
	lines := make([]zoneLine, 0, len(z.Nameservers)+len(z.Records))
	for _, ns := range z.Nameservers {
		lines = append(lines, zoneLine{owner: "@", rtype: "NS", data: config.NormalizeDNSName(ns)})
	}

	for _, r := range z.Records {
		lines = append(lines, zoneLine{
			owner: strings.TrimSpace(r.Name),
			ttl:   recordTTL(r.TTL),
			rtype: strings.ToUpper(strings.TrimSpace(r.Type)),
			data:  recordData(r),
		})
	}

	return lines
}

func zoneSerial(z config.DNSZone, soa soaValues, lines []zoneLine) string {
	if soa.Serial > 0 {
		return strconv.Itoa(soa.Serial)
	}

	h := fnv.New32a()
	write := func(parts ...string) { zoneSerialCallback(h, parts...) }

	write(config.NormalizeDNSName(z.Name), strconv.Itoa(zoneTTL(z)), soa.Primary, soa.Email,
		strconv.Itoa(soa.Refresh), strconv.Itoa(soa.Retry), strconv.Itoa(soa.Expire), strconv.Itoa(soa.Minimum))

	for _, l := range lines {
		write(l.owner, l.ttl, l.rtype, l.data)
	}

	return strconv.FormatUint(uint64(h.Sum32()), 10)
}

func formatZoneLines(lines []zoneLine) []string {
	var ownerWidth, ttlWidth, typeWidth int
	for _, l := range lines {
		ownerWidth = max(ownerWidth, len(l.owner))
		ttlWidth = max(ttlWidth, len(l.ttl))
		typeWidth = max(typeWidth, len(l.rtype))
	}

	out := make([]string, 0, len(lines))
	for _, l := range lines {
		var b strings.Builder
		_, _ = fmt.Fprintf(&b, "%-*s ", ownerWidth, l.owner)
		if ttlWidth > 0 {
			_, _ = fmt.Fprintf(&b, "%-*s ", ttlWidth, l.ttl)
		}

		_, _ = fmt.Fprintf(&b, "IN %-*s %s", typeWidth, l.rtype, l.data)
		out = append(out, strings.TrimRight(b.String(), " "))
	}

	return out
}

func renderZoneFile(z config.DNSZone) string {
	return renderZoneFileWith(z, "", nil)
}

func renderZoneFileWith(z config.DNSZone, serial string, extra []zoneLine) string {
	origin := config.NormalizeDNSName(z.Name)
	soa := zoneSOA(z)
	lines := zoneLines(z)
	if serial == "" {
		serial = zoneSerial(z, soa, lines)
	}

	soaData := fmt.Sprintf("%s %s ( %s %d %d %d %d )",
		soa.Primary, soa.Email, serial, soa.Refresh, soa.Retry, soa.Expire, soa.Minimum)

	all := append([]zoneLine{{owner: "@", rtype: "SOA", data: soaData}}, lines...)
	all = append(all, extra...)

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "; routier:%s\n", origin)
	_, _ = fmt.Fprintf(&b, "$ORIGIN %s\n", origin)
	_, _ = fmt.Fprintf(&b, "$TTL %d\n", zoneTTL(z))

	for _, l := range formatZoneLines(all) {
		_, _ = b.WriteString(l + "\n")
	}

	return b.String()
}

func ddnsActive(cfg *config.Config) bool {
	return LocalKeaDDNSEnabled(cfg) && LocalDNSServerEnabled(cfg)
}

func reverseZoneName(cidr string) (string, bool) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", false
	}

	ones, _ := ipnet.Mask.Size()
	if ip4 := ipnet.IP.To4(); ip4 != nil {
		if ones == 0 || ones > 24 || ones%8 != 0 {
			return "", false
		}

		parts := make([]string, 0, ones/8)
		for i := ones/8 - 1; i >= 0; i-- {
			parts = append(parts, strconv.Itoa(int(ip4[i])))
		}

		return strings.Join(parts, ".") + ".in-addr.arpa", true
	}

	if ones == 0 || ones%4 != 0 {
		return "", false
	}

	ip16 := ipnet.IP.To16()
	if ip16 == nil {
		return "", false
	}

	parts := make([]string, 0, ones/4)
	for i := ones/4 - 1; i >= 0; i-- {
		nib := ip16[i/2] >> 4
		if i%2 == 1 {
			nib = ip16[i/2] & 0x0f
		}

		parts = append(parts, strconv.FormatUint(uint64(nib), 16))
	}

	return strings.Join(parts, ".") + ".ip6.arpa", true
}

func ddnsSubnetUpdates(s config.KeaSubnet) bool {
	return s.DDNS == nil || *s.DDNS
}

func ddnsForwardZone(cfg *config.Config) string {
	if !ddnsActive(cfg) {
		return ""
	}

	return strings.TrimSuffix(cfg.DHCP.DDNS.Domain, ".")
}

func ddnsForwardUpdateZone(cfg *config.Config, domain string) string {
	name := strings.TrimSuffix(config.NormalizeDNSName(domain), ".")
	best := ""
	for _, zone := range cfg.DNS.Server.Zones {
		candidate := strings.TrimSuffix(config.NormalizeDNSName(zone.Name), ".")
		if candidate != "" && (name == candidate || strings.HasSuffix(name, "."+candidate)) && len(candidate) > len(best) {
			best = candidate
		}
	}

	if best != "" {
		return best
	}

	return name
}

func ddnsForwardZones(cfg *config.Config) []string {
	if !ddnsActive(cfg) {
		return nil
	}

	seen := make(map[string]bool)
	var names []string
	add := func(domain string) { ddnsForwardZonesCallback(cfg, seen, &names, domain) }

	add(cfg.DHCP.DDNS.Domain)
	for _, list := range [][]config.KeaSubnet{cfg.DHCP.Subnets4, cfg.DHCP.Subnets6} {
		for _, s := range list {
			if ddnsSubnetUpdates(s) {
				add(s.DDNSDomain)
			}
		}
	}

	return names
}

func ddnsReverseZoneNames(cfg *config.Config) []string {
	if !ddnsActive(cfg) || !cfg.DHCP.DDNS.UpdatesReverse() {
		return nil
	}

	seen := make(map[string]bool)
	var names []string
	for _, list := range [][]config.KeaSubnet{cfg.DHCP.Subnets4, cfg.DHCP.Subnets6} {
		for _, s := range list {
			if !ddnsSubnetUpdates(s) {
				continue
			}

			name, ok := reverseZoneName(s.Subnet)
			if !ok {
				log.Warn().Str("subnet", s.Subnet).Msg("ddns: reverse zone not derivable (non-aligned prefix), PTR updates skipped")
				continue
			}

			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}

	return names
}

func ddnsZoneNames(cfg *config.Config) []string {
	fwd := ddnsForwardZones(cfg)
	if len(fwd) == 0 {
		return nil
	}

	return append(fwd, ddnsReverseZoneNames(cfg)...)
}

func ddnsManagedZone(cfg *config.Config, name string) bool {
	for _, candidate := range ddnsZoneNames(cfg) {
		if strings.EqualFold(config.NormalizeDNSName(candidate), config.NormalizeDNSName(name)) {
			return true
		}
	}

	return false
}

func ddnsUndeclaredZones(cfg *config.Config) []string {
	var names []string
	for _, name := range ddnsZoneNames(cfg) {
		declared := false
		for _, zone := range cfg.DNS.Server.Zones {
			if strings.EqualFold(config.NormalizeDNSName(zone.Name), config.NormalizeDNSName(name)) {
				declared = true
				break
			}
		}

		if !declared {
			names = append(names, name)
		}
	}

	return names
}

func dnsServerPort(cfg *config.Config) int {
	if s := dnsServer(cfg); s != nil && s.Port > 0 {
		return s.Port
	}

	return 53
}

func bindTSIGAlgorithm(cfg *config.Config) string {
	if !ddnsActive(cfg) {
		return ""
	}

	if a := cfg.DHCP.DDNS.Algorithm; a != "" {
		return a
	}

	return config.DefaultDDNSAlgorithm
}

func ddnsKeySecret(cfg *config.Config) string {
	if !ddnsActive(cfg) {
		return ""
	}

	return cfg.DHCP.DDNS.Key
}

func DDNSActive(cfg *config.Config) bool {
	return ddnsActive(cfg)
}

func DDNSZoneNames(cfg *config.Config) []string {
	return ddnsZoneNames(cfg)
}

func DDNSBootstrapZoneFile(cfg *config.Config, name string) string {
	for _, zone := range cfg.DNS.Server.Zones {
		if strings.EqualFold(config.NormalizeDNSName(zone.Name), config.NormalizeDNSName(name)) {
			return renderZoneFile(zone)
		}
	}

	return renderZoneFile(ddnsBootstrapZone(cfg, name))
}

func ddnsBootstrapZone(cfg *config.Config, name string) config.DNSZone {
	ns := "ns." + strings.TrimSuffix(cfg.DHCP.DDNS.Domain, ".")
	ttl := cfg.DHCP.DDNS.TTL
	if ttl <= 0 {
		ttl = defaultZoneTTL
	}

	return config.DNSZone{
		Name:        name,
		TTL:         ttl,
		Nameservers: []string{ns},
	}
}

func renderDNSZones(cfg *config.Config) ([]Output, error) {
	s := dnsServer(cfg)
	if s == nil {
		return nil, nil
	}

	zones := append([]config.DNSZone{}, s.Zones...)
	for _, v := range s.Views {
		zones = append(zones, v.Zones...)
	}

	var out []Output
	for _, z := range zones {
		if ddnsManagedZone(cfg, z.Name) {
			if len(z.Primaries) > 0 {
				return nil, fmt.Errorf("DDNS zone %q must be a primary zone", z.Name)
			}

			if len(s.Views) > 0 {
				return nil, fmt.Errorf("DDNS zones in DNS views are not supported")
			}

			if len(z.Records) > 0 {
				out = append(out, Output{
					Name:    namedZonePfx + ZoneFileName(z.Name),
					Dest:    staticShadowPath(z.Name),
					Content: renderZoneFile(z),
				})
			}

			continue
		}

		if len(z.Records) == 0 {
			continue
		}

		out = append(out, Output{
			Name:    namedZonePfx + ZoneFileName(z.Name),
			Dest:    zoneFilePath(z.Name),
			Content: renderZoneFile(z),
		})
	}

	return out, nil
}

func sortIPStringsCallback(addrs []string, i, j int) bool {
	a, b := net.ParseIP(addrs[i]), net.ParseIP(addrs[j])
	if a == nil || b == nil {
		return addrs[i] < addrs[j]
	}

	a4, b4 := a.To4() != nil, b.To4() != nil
	if a4 != b4 {
		return a4
	}

	return bytes.Compare(a.To16(), b.To16()) < 0
}

func dnsInsecureDomainsCallback(names *[]string, seen map[string]bool, raw string, signed bool) {
	name := strings.TrimSuffix(config.NormalizeDNSName(raw), ".")
	if signed || name == "" || seen[name] {
		return
	}

	seen[name] = true
	(*names) = append((*names), name)
}

func zoneSerialCallback(h hash.Hash32, parts ...string) {
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
}

func ddnsForwardZonesCallback(cfg *config.Config, seen map[string]bool, names *[]string, domain string) {
	name := ddnsForwardUpdateZone(cfg, domain)
	if name == "" || seen[name] {
		return
	}

	seen[name] = true
	(*names) = append((*names), name)
}
