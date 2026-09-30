package simple

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/config/layers"
	"github.com/ChevalRouting/routier/pkg/nftables/nat"
)

func inspect(cfg *config.Config) Config {
	result := Config{Networks: []Network{}, System: System{Hostname: cfg.Hostname}, Users: inspectUsers(cfg.Users), Monitoring: cfg.Monitoring}
	internetName := findInternet(cfg)
	if internetName != "" {
		iface := cfg.Interfaces[internetName]
		result.Internet = &Internet{
			Interface: internetName,
			Select:    iface.Select,
			Mode:      addressMode(iface.Addresses),
			IPv6:      ipv6Mode(iface.Addresses),
			Addresses: staticAddresses(iface.Addresses),
			Gateway:   defaultGateway(cfg, internetName, "0.0.0.0/0"),
			GatewayV6: defaultGateway(cfg, internetName, "::/0"),
			VIPs:      internetVIPs(cfg, internetName),
		}
		if cfg.DNS != nil {
			result.Internet.DNS = append([]string(nil), cfg.DNS.Nameservers...)
		}
	}

	names := make([]string, 0, len(cfg.Interfaces))
	for name := range cfg.Interfaces {
		if name != internetName {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	for _, name := range names {
		network, issue := inspectNetwork(cfg, name)
		if network == nil {
			continue
		}

		result.Networks = append(result.Networks, *network)
		if issue != nil {
			result.Issues = append(result.Issues, *issue)
		}
	}

	result.PortForwards = inspectPortForwards(cfg)
	result.DNS = inspectDNS(cfg, result.Networks)

	return result
}

func inspectUsers(users map[string]*config.User) map[string]*config.User {
	result := make(map[string]*config.User, len(users))
	for name, user := range users {
		if user == nil {
			result[name] = nil
			continue
		}

		copy := *user
		copy.Groups = append([]string(nil), user.Groups...)
		copy.SSHKeys = append([]string(nil), user.SSHKeys...)
		result[name] = &copy
	}

	return result
}

func inspectDNS(cfg *config.Config, networks []Network) DNS {
	result := DNS{Upstreams: []string{}, Networks: []string{}, WANAllowFrom: []string{}, Zones: []config.DNSZone{}, Cache: true}
	if cfg.DNS == nil || cfg.DNS.Server == nil {
		return result
	}

	server := cfg.DNS.Server
	result.Zones = append([]config.DNSZone(nil), server.Zones...)
	result.Enabled = server.Enabled
	result.Upstreams = append([]string(nil), server.Upstreams...)
	result.DNSSEC = server.DNSSEC
	result.Cache = server.Cache == nil || !server.Cache.Disabled
	if server.Cache != nil {
		result.Prefetch = server.Cache.Prefetch
		result.ServeExpired = server.Cache.ServeExpired
	}

	listens := make(map[string]bool, len(server.Listen))
	for _, value := range server.Listen {
		listens[value] = true
	}
	for _, network := range networks {
		if listens["iface("+network.Name+")"] {
			result.Networks = append(result.Networks, network.Name)
		}
	}
	internet := findInternet(cfg)
	result.AllowWAN = internet != "" && listens["iface("+internet+")"]
	if result.AllowWAN {
		local := map[string]bool{"127.0.0.0/8": true, "::1/128": true}
		for _, network := range networks {
			if !listens["iface("+network.Name+")"] {
				continue
			}
			for _, address := range network.Addresses {
				prefix, err := netip.ParsePrefix(address)
				if err == nil {
					local[prefix.Masked().String()] = true
				}
			}
		}
		for _, source := range server.AllowFrom {
			if !local[source] {
				result.WANAllowFrom = append(result.WANAllowFrom, source)
			}
		}
	}

	return result
}

func nftName(name string) string {
	return strings.NewReplacer(":", "_", ".", "_", "-", "_").Replace(name)
}

func networkSource(name string) string {
	return "$" + nftName(name) + "_network"
}

func internetInterfaceSet(cfg *config.Config) string {
	name := findInternet(cfg)
	if name == "" {
		return ""
	}

	return "$" + nftName(name) + "_interfaces"
}

func networkMasquerades(cfg *config.Config, name string) bool {
	source := networkSource(name)
	for _, spec := range nat.Parse(cfg.Nftables) {
		if spec.Kind == nat.KindMasquerade && spec.Source == source {
			return true
		}
	}

	return false
}

func inspectPortForwards(cfg *config.Config) []PortForward {
	internet := internetInterfaceSet(cfg)
	forwards := []PortForward{}
	paired := map[string]int{}
	index := 0
	for _, spec := range nat.Parse(cfg.Nftables) {
		if spec.Kind != nat.KindDNAT {
			continue
		}

		index++
		host, port := splitHostPort(spec.To)
		forward := PortForward{
			ID:       fmt.Sprintf("forward-%d", index),
			Name:     spec.Comment,
			Proto:    spec.Proto,
			Port:     spec.DPort,
			ToHost:   host,
			ToPort:   port,
			Editable: true,
		}
		if spec.In != "" && spec.In != internet {
			forward.Editable = false
			forward.Issue = "This port forward uses a custom interface. Manage it in Advanced mode."
		}
		if spec.Proto != "tcp" && spec.Proto != "udp" {
			forward.Editable = false
			forward.Issue = "This port forward uses a custom protocol. Manage it in Advanced mode."
		}

		key := strings.Join([]string{spec.In, spec.DPort, spec.To, spec.Comment}, "\x00")
		if previous, ok := paired[key]; ok && forward.Editable && forwards[previous].Editable && forwards[previous].Proto != spec.Proto {
			forwards[previous].Proto = "tcp+udp"
			continue
		}

		paired[key] = len(forwards)
		forwards = append(forwards, forward)
	}

	return forwards
}

func splitHostPort(to string) (string, string) {
	if to == "" {
		return "", ""
	}

	if strings.HasPrefix(to, "[") {
		if end := strings.LastIndex(to, "]"); end >= 0 {
			host := to[1:end]
			if rest := to[end+1:]; strings.HasPrefix(rest, ":") {
				return host, rest[1:]
			}

			return host, ""
		}
	}

	if i := strings.LastIndex(to, ":"); i >= 0 && !strings.Contains(to[:i], ":") {
		return to[:i], to[i+1:]
	}

	return to, ""
}

func findInternet(cfg *config.Config) string {
	if iface := cfg.Interfaces["wan"]; iface != nil {
		return "wan"
	}

	if cfg.Routing != nil {
		for _, route := range cfg.Routing.Static {
			if route.Destination == "0.0.0.0/0" && route.Dev != "" && cfg.Interfaces[route.Dev] != nil {
				return route.Dev
			}
		}
	}

	names := make([]string, 0, len(cfg.Interfaces))
	for name := range cfg.Interfaces {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		iface := cfg.Interfaces[name]
		if addressMode(iface.Addresses) == "dhcp" {
			return name
		}
	}

	return ""
}

func inspectNetwork(cfg *config.Config, name string) (*Network, *layers.Issue) {
	iface := cfg.Interfaces[name]
	if iface == nil || iface.Type != "" || iface.Bridge != nil || iface.VXLAN != nil {
		return nil, nil
	}

	static := staticAddresses(iface.Addresses)
	if len(static) == 0 {
		return nil, nil
	}

	network := &Network{ID: name, Name: name, Select: iface.Select, Addresses: static, Masquerade: networkMasquerades(cfg, name), Editable: true}
	if reasons := advancedReasons(cfg, name, iface); len(reasons) != 0 {
		network.Editable = false
		network.Issue = fmt.Sprintf("%q uses %s, which the simple view cannot edit safely. Manage it in Advanced mode.", name, joinReasons(reasons))
		issue := &layers.Issue{
			Section: "networks",
			Paths:   []string{"interfaces." + name},
			Message: network.Issue,
		}
		return network, issue
	}

	v4 := staticIPv4(iface.Addresses)
	if len(v4) > 0 {
		subnet := dhcpSubnet(cfg, name, v4[0])
		if subnet != nil {
			network.ManageDHCP = true
			if len(subnet.Pools) > 0 {
				network.Pool = subnet.Pools[0]
			}
			network.Gateway = subnet.Gateway
			network.DNS = append([]string(nil), subnet.DNS...)
			network.Exclusions = append([]string(nil), subnet.Exclusions...)
			network.ValidLifetime = subnet.ValidLifetime
		}
	}

	v6 := staticIPv6(iface.Addresses)
	if len(v6) > 0 && cfg.DHCP != nil {
		if subnet6 := dhcpSubnetFamily(cfg.DHCP.Subnets6, name, v6[0]); subnet6 != nil {
			network.ManageDHCP6 = true
			if len(subnet6.Pools) > 0 {
				network.PoolV6 = subnet6.Pools[0]
			}
			network.ValidLifetimeV6 = subnet6.ValidLifetime
		}
	}

	return network, nil
}

func internetVIPs(cfg *config.Config, name string) []string {
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

func hasVRRP(cfg *config.Config, name string) bool {
	if cfg.HA == nil {
		return false
	}

	for _, v := range cfg.HA.VRRP {
		if v.Interface == name {
			return true
		}
	}

	return false
}

func advancedReasons(cfg *config.Config, name string, iface *config.Interface) []string {
	var reasons []string

	if iface.VRF != "" {
		reasons = append(reasons, "a VRF")
	}

	if hasVRRP(cfg, name) {
		reasons = append(reasons, "VRRP failover")
	}

	return reasons
}

func joinReasons(reasons []string) string {
	switch len(reasons) {
	case 0:
		return ""
	case 1:
		return reasons[0]
	case 2:
		return reasons[0] + " and " + reasons[1]
	default:
		return strings.Join(reasons[:len(reasons)-1], ", ") + " and " + reasons[len(reasons)-1]
	}
}

func dhcpSubnet(cfg *config.Config, name, address string) *config.KeaSubnet {
	if cfg.DHCP == nil {
		return nil
	}

	ip, err := netip.ParsePrefix(address)
	if err != nil {
		return nil
	}

	return dhcpSubnetFamily(cfg.DHCP.Subnets4, name, ip.String())
}

func dhcpSubnetFamily(subnets []config.KeaSubnet, name, address string) *config.KeaSubnet {
	ip, err := netip.ParsePrefix(address)
	if err != nil {
		return nil
	}
	want := ip.Masked()
	for i := range subnets {
		subnet := &subnets[i]
		prefix, err := netip.ParsePrefix(subnet.Subnet)
		if err == nil && subnet.Interface == name && prefix.Masked() == want {
			return subnet
		}
	}

	return nil
}

func addressMode(addresses []string) string {
	for _, address := range addresses {
		if address == "dhcp" || address == "dhcp4" {
			return "dhcp"
		}
	}

	if len(staticIPv4(addresses)) > 0 {
		return "static"
	}

	return "disabled"
}

func staticIPv4(addresses []string) []string {
	var values []string
	for _, address := range addresses {
		prefix, err := netip.ParsePrefix(address)
		if err == nil && prefix.Addr().Is4() {
			values = append(values, address)
		}
	}

	return values
}

func staticIPv6(addresses []string) []string {
	var values []string
	for _, address := range addresses {
		prefix, err := netip.ParsePrefix(address)
		if err == nil && prefix.Addr().Is6() {
			values = append(values, address)
		}
	}

	return values
}

func ipv6Mode(addresses []string) string {
	for _, address := range addresses {
		if address == "slaac" {
			return "slaac"
		}
	}

	for _, address := range addresses {
		if address == "dhcp6" {
			return "dhcp6"
		}
	}

	if len(staticIPv6(addresses)) > 0 {
		return "static"
	}

	return "disabled"
}

func staticAddresses(addresses []string) []string {
	var values []string
	for _, address := range addresses {
		if _, err := netip.ParsePrefix(address); err == nil {
			values = append(values, address)
		}
	}

	return values
}

func defaultGateway(cfg *config.Config, iface, destination string) string {
	if cfg.Routing == nil {
		return ""
	}

	for _, route := range cfg.Routing.Static {
		if route.Destination == destination && (route.Dev == "" || route.Dev == iface) {
			return route.Via
		}
	}

	return ""
}

func networkPrefix(address string) (string, error) {
	prefix, err := netip.ParsePrefix(address)
	if err != nil || !prefix.Addr().Is4() {
		return "", fmt.Errorf("address must be an IPv4 CIDR")
	}

	return prefix.Masked().String(), nil
}

func simpleTag(name string) string {
	return "simple:network:" + strings.ToLower(name)
}
