package simple

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/nftables/nat"
)

func replaceInternet(cfg *config.Config, value *Internet) (*config.Config, error) {
	previous := findInternet(cfg)
	if value == nil {
		if previous != "" {
			delete(cfg.Interfaces, previous)
		}
		removeDefaultRoutes(cfg)
		return cfg, nil
	}

	if value.Interface == "" || value.Select == "" {
		return nil, fmt.Errorf("simple internet requires an interface and selector")
	}

	if len(staticAddresses(value.Addresses)) != len(value.Addresses) {
		return nil, fmt.Errorf("simple internet addresses must be IP CIDRs")
	}

	var addresses []string
	switch value.Mode {
	case "dhcp":
		addresses = append(addresses, "dhcp")
	case "static":
		v4 := staticIPv4(value.Addresses)
		if len(v4) == 0 {
			return nil, fmt.Errorf("simple static internet requires an IPv4 address")
		}
		addresses = append(addresses, v4...)
	case "disabled":
	default:
		return nil, fmt.Errorf("unknown simple internet mode: %s", value.Mode)
	}

	switch value.IPv6 {
	case "slaac":
		addresses = append(addresses, "slaac")
	case "dhcp6":
		addresses = append(addresses, "dhcp6")
	case "static":
		v6 := staticIPv6(value.Addresses)
		if len(v6) == 0 {
			return nil, fmt.Errorf("simple static IPv6 internet requires an IPv6 address")
		}
		addresses = append(addresses, v6...)
	case "disabled", "":
	default:
		return nil, fmt.Errorf("unknown simple internet ipv6 mode: %s", value.IPv6)
	}

	if cfg.Interfaces == nil {
		cfg.Interfaces = map[string]*config.Interface{}
	}
	iface := cfg.Interfaces[previous]
	if iface == nil {
		iface = &config.Interface{}
	}
	if previous != "" && previous != value.Interface {
		delete(cfg.Interfaces, previous)
	}
	iface.Select = value.Select
	iface.Addresses = addresses
	cfg.Interfaces[value.Interface] = iface
	removeDefaultRoutes(cfg)
	if value.Mode == "static" && value.Gateway != "" {
		gateway, err := netip.ParseAddr(value.Gateway)
		if err != nil || !gateway.Is4() {
			return nil, fmt.Errorf("simple internet gateway must be an IPv4 address: %s", value.Gateway)
		}
		addDefaultRoute(cfg, "0.0.0.0/0", value.Gateway, value.Interface)
	}
	if value.IPv6 == "static" && value.GatewayV6 != "" {
		gateway, err := netip.ParseAddr(value.GatewayV6)
		if err != nil || !gateway.Is6() {
			return nil, fmt.Errorf("simple internet IPv6 gateway must be an IPv6 address: %s", value.GatewayV6)
		}
		addDefaultRoute(cfg, "::/0", value.GatewayV6, value.Interface)
	}
	if cfg.DNS == nil {
		cfg.DNS = &config.DNS{}
	}
	cfg.DNS.Nameservers = append([]string(nil), value.DNS...)

	return cfg, nil
}

func replaceNetworks(cfg *config.Config, values []Network) (*config.Config, error) {
	current := inspect(cfg)
	interfaces := map[string]*config.Interface{}
	subnets := map[string]config.KeaSubnet{}
	subnets6 := map[string]config.KeaSubnet{}
	locked := map[string]bool{}
	if cfg.Interfaces == nil {
		cfg.Interfaces = map[string]*config.Interface{}
	}

	for _, network := range current.Networks {
		if !network.Editable {
			locked[network.Name] = true
			continue
		}
		interfaces[network.Name] = cfg.Interfaces[network.Name]
		delete(cfg.Interfaces, network.Name)
	}

	if cfg.DHCP != nil {
		kept := cfg.DHCP.Subnets4[:0]
		for _, subnet := range cfg.DHCP.Subnets4 {
			if hasNetwork(current.Networks, subnet.Interface) {
				subnets[subnet.Interface] = subnet
			} else {
				kept = append(kept, subnet)
			}
		}
		cfg.DHCP.Subnets4 = kept
		kept6 := cfg.DHCP.Subnets6[:0]
		for _, subnet := range cfg.DHCP.Subnets6 {
			if hasNetwork(current.Networks, subnet.Interface) {
				subnets6[subnet.Interface] = subnet
			} else {
				kept6 = append(kept6, subnet)
			}
		}
		cfg.DHCP.Subnets6 = kept6
	}
	removeSimpleRADVD(cfg, current.Networks)

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	seen := map[string]bool{}
	for _, value := range values {
		id := value.ID
		if id == "" {
			id = value.Name
		}
		if locked[id] {
			continue
		}
		if value.Name == "" || value.Select == "" {
			return nil, fmt.Errorf("simple networks require a name and selector")
		}
		if seen[value.Name] {
			return nil, fmt.Errorf("simple network name %s is duplicated", value.Name)
		}
		seen[value.Name] = true
		if value.Name != id && cfg.Interfaces[value.Name] != nil {
			return nil, fmt.Errorf("interface %s is managed by advanced configuration", value.Name)
		}
		addresses := staticAddresses(value.Addresses)
		if len(addresses) == 0 || len(addresses) != len(value.Addresses) {
			return nil, fmt.Errorf("simple network %s: each address must be an IP CIDR", value.Name)
		}

		iface := interfaces[id]
		if iface == nil {
			iface = &config.Interface{}
		}
		iface.Select = value.Select
		iface.Addresses = addresses
		cfg.Interfaces[value.Name] = iface
		if value.ManageDHCP {
			v4 := staticIPv4(addresses)
			if len(v4) == 0 {
				return nil, fmt.Errorf("simple network %s: DHCPv4 needs an IPv4 address", value.Name)
			}
			prefix, err := networkPrefix(v4[0])
			if err != nil {
				return nil, fmt.Errorf("simple network %s: %w", value.Name, err)
			}
			if value.Pool == "" {
				return nil, fmt.Errorf("simple network %s requires a DHCPv4 pool", value.Name)
			}
			if cfg.DHCP == nil {
				cfg.DHCP = &config.DHCP{}
			}
			gateway, _ := netip.ParsePrefix(v4[0])
			subnet := subnets[id]
			subnet.Subnet, subnet.Interface, subnet.Pools = prefix, value.Name, []string{value.Pool}
			subnet.Gateway = value.Gateway
			if subnet.Gateway == "" {
				subnet.Gateway = gateway.Addr().String()
			}
			subnet.DNS = append([]string(nil), value.DNS...)
			subnet.Exclusions = append([]string(nil), value.Exclusions...)
			subnet.ValidLifetime = value.ValidLifetime
			cfg.DHCP.Subnets4 = append(cfg.DHCP.Subnets4, subnet)
		}
		if value.ManageDHCP6 {
			v6 := staticIPv6(addresses)
			if len(v6) == 0 {
				return nil, fmt.Errorf("simple network %s: DHCPv6 needs an IPv6 prefix", value.Name)
			}
			prefix, err := netip.ParsePrefix(v6[0])
			if err != nil {
				return nil, fmt.Errorf("simple network %s: invalid IPv6 prefix", value.Name)
			}
			if value.PoolV6 == "" {
				return nil, fmt.Errorf("simple network %s requires a DHCPv6 pool", value.Name)
			}
			if cfg.DHCP == nil {
				cfg.DHCP = &config.DHCP{}
			}
			subnet := subnets6[id]
			subnet.Subnet, subnet.Interface, subnet.Pools = prefix.Masked().String(), value.Name, []string{value.PoolV6}
			subnet.DNS = append([]string(nil), value.DNS...)
			subnet.ValidLifetime = value.ValidLifetimeV6
			cfg.DHCP.Subnets6 = append(cfg.DHCP.Subnets6, subnet)
			addSimpleRADVD(cfg, value.Name, prefix.Masked().String(), value.DNS)
		}
	}
	if cfg.DHCP != nil {
		cfg.DHCP.Enabled = len(cfg.DHCP.Subnets4) > 0 || len(cfg.DHCP.Subnets6) > 0
	}

	syncMasquerade(cfg, current.Networks, values)

	return cfg, nil
}

func removeSimpleRADVD(cfg *config.Config, networks []Network) {
	if cfg.Routing == nil || cfg.Routing.RADVD == nil {
		return
	}
	for _, network := range networks {
		if network.Editable {
			delete(cfg.Routing.RADVD.Interfaces, network.Name)
		}
	}
	if len(cfg.Routing.RADVD.Interfaces) == 0 {
		cfg.Routing.RADVD = nil
	}
}

func addSimpleRADVD(cfg *config.Config, name, prefix string, dns []string) {
	if cfg.Routing == nil {
		cfg.Routing = &config.Routing{}
	}
	if cfg.Routing.RADVD == nil {
		cfg.Routing.RADVD = &config.RADVDConfig{Interfaces: map[string]*config.RADVDInterface{}}
	}
	if cfg.Routing.RADVD.Interfaces == nil {
		cfg.Routing.RADVD.Interfaces = map[string]*config.RADVDInterface{}
	}
	rdns := make([]string, 0, len(dns))
	for _, address := range dns {
		if ip, err := netip.ParseAddr(address); err == nil && ip.Is6() {
			rdns = append(rdns, address)
		}
	}
	entry := &config.RADVDInterface{AdvSendAdvert: true, AdvManagedFlag: true, AdvOtherConfigFlag: true,
		Prefixes: []config.RADVDPrefix{{Prefix: prefix, AdvOnLink: true}}}
	if len(rdns) > 0 {
		entry.RDNSS = &config.RADVDRDNSS{Servers: rdns}
	}
	cfg.Routing.RADVD.Interfaces[name] = entry
}

func syncMasquerade(cfg *config.Config, current, values []Network) {
	wanted := false
	for _, network := range values {
		if network.Masquerade && len(staticIPv4(network.Addresses)) > 0 {
			wanted = true
			break
		}
	}
	if cfg.Nftables == nil && !wanted {
		return
	}

	owned := map[string]bool{}
	for _, network := range current {
		owned[networkSource(network.Name)] = true
	}
	for _, network := range values {
		owned[networkSource(network.Name)] = true
	}

	var kept []nat.Spec
	for _, spec := range nat.Parse(cfg.Nftables) {
		if spec.Kind == nat.KindMasquerade && owned[spec.Source] {
			continue
		}

		kept = append(kept, spec)
	}

	out := internetInterfaceSet(cfg)
	for _, network := range values {
		if !network.Masquerade || len(staticIPv4(network.Addresses)) == 0 {
			continue
		}

		kept = append(kept, nat.Spec{
			Kind:   nat.KindMasquerade,
			Source: networkSource(network.Name),
			Out:    out,
			Family: "ip",
		})
	}

	nat.Apply(cfg, kept)
}

const simpleFirewallTag = "simple"

func syncFirewallDefaults(cfg *config.Config, internet *Internet, networks []Network) {
	if cfg.Nftables == nil {
		cfg.Nftables = &config.NftablesConfig{}
	}
	if cfg.Nftables.Chains == nil {
		cfg.Nftables.Chains = map[string]*config.NftChain{}
	}

	setRules := func(chain string, generated []config.ManagedRule) {
		current := cfg.Nftables.Chains[chain]
		if current == nil {
			current = &config.NftChain{}
			cfg.Nftables.Chains[chain] = current
		}

		kept := current.Managed[:0]
		for _, rule := range current.Managed {
			if rule.Tag != simpleFirewallTag {
				kept = append(kept, rule)
			}
		}
		current.Managed = append(kept, generated...)
	}

	setRules("input", []config.ManagedRule{
		{Tag: simpleFirewallTag, Comment: "allow loopback", Match: &config.RuleMatch{IIF: "lo"}, Action: "accept"},
		{Tag: simpleFirewallTag, Comment: "allow established traffic", Match: &config.RuleMatch{CTState: "established,related"}, Action: "accept"},
		{Tag: simpleFirewallTag, Comment: "drop invalid traffic", Match: &config.RuleMatch{CTState: "invalid"}, Action: "drop"},
		{Tag: simpleFirewallTag, Comment: "allow IPv4 ICMP", Match: &config.RuleMatch{Protocol: "icmp"}, Action: "accept"},
		{Tag: simpleFirewallTag, Comment: "allow IPv6 neighbor discovery", Match: &config.RuleMatch{Protocol: "icmpv6"}, Action: "accept"},
		{Tag: simpleFirewallTag, Comment: "allow administration", Match: &config.RuleMatch{Protocol: "tcp", DPort: "ssh, 8080"}, Action: "accept"},
	})

	forward := []config.ManagedRule{
		{Tag: simpleFirewallTag, Comment: "allow established forwarding", Match: &config.RuleMatch{CTState: "established,related"}, Action: "accept"},
		{Tag: simpleFirewallTag, Comment: "drop invalid forwarding", Match: &config.RuleMatch{CTState: "invalid"}, Action: "drop"},
	}
	if internet != nil {
		out := "$" + nftName(internet.Interface) + "_interfaces"
		for _, network := range networks {
			if !network.Masquerade || len(staticIPv4(network.Addresses)) == 0 {
				continue
			}
			forward = append(forward, config.ManagedRule{
				Tag:     simpleFirewallTag,
				Comment: "allow " + network.Name + " to Internet",
				Match: &config.RuleMatch{
					IIF: "$" + nftName(network.Name) + "_interfaces",
					OIF: out,
				},
				Action: "accept",
			})
		}
	}
	setRules("forward", forward)
}

func replacePortForwards(cfg *config.Config, values []PortForward) (*config.Config, error) {
	wanted := false
	for _, forward := range values {
		if forward.Editable {
			wanted = true
			break
		}
	}
	if cfg.Nftables == nil && !wanted {
		return cfg, nil
	}

	internet := internetInterfaceSet(cfg)

	var kept []nat.Spec
	for _, spec := range nat.Parse(cfg.Nftables) {
		if spec.Kind != nat.KindDNAT {
			kept = append(kept, spec)
			continue
		}

		if spec.In != "" && spec.In != internet {
			kept = append(kept, spec)
		}
	}

	for _, forward := range values {
		if !forward.Editable {
			continue
		}
		if forward.Port == "" || forward.ToHost == "" {
			return nil, fmt.Errorf("port forward %q needs an external port and a destination host", forward.Name)
		}

		proto := forward.Proto
		if proto == "" {
			proto = "tcp"
		}
		if proto != "tcp" && proto != "udp" && proto != "tcp+udp" {
			return nil, fmt.Errorf("port forward %q protocol must be tcp, udp or tcp+udp", forward.Name)
		}

		to := forward.ToHost
		if forward.ToPort != "" {
			to = joinHostPort(forward.ToHost, forward.ToPort)
		}

		protocols := []string{proto}
		if proto == "tcp+udp" {
			protocols = []string{"tcp", "udp"}
		}
		for _, protocol := range protocols {
			kept = append(kept, nat.Spec{
				Kind:    nat.KindDNAT,
				In:      internet,
				Proto:   protocol,
				DPort:   forward.Port,
				To:      to,
				Comment: forward.Name,
			})
		}
	}

	nat.Apply(cfg, kept)
	return cfg, nil
}

func replaceDNS(cfg *config.Config, value DNS) (*config.Config, error) {
	if cfg.DNS == nil {
		cfg.DNS = &config.DNS{}
	}
	if !value.Enabled {
		cfg.DNS.Server = nil
		return cfg, nil
	}
	if len(value.Upstreams) == 0 {
		return nil, fmt.Errorf("simple dns requires at least one upstream resolver")
	}

	server := &config.DNSServer{
		Enabled:   true,
		Mode:      config.DNSModeForwarder,
		Listen:    []string{"127.0.0.1", "::1"},
		AllowFrom: []string{"127.0.0.0/8", "::1/128"},
		Upstreams: append([]string(nil), value.Upstreams...),
		DNSSEC:    value.DNSSEC,
		Zones:     append([]config.DNSZone(nil), value.Zones...),
		Cache: &config.DNSCache{
			Disabled:     !value.Cache,
			Prefetch:     value.Prefetch,
			ServeExpired: value.ServeExpired,
		},
	}
	for _, name := range value.Networks {
		iface := cfg.Interfaces[name]
		if iface == nil {
			return nil, fmt.Errorf("simple dns network %s does not exist", name)
		}
		server.Listen = append(server.Listen, "iface("+name+")")
		server.AllowInbound = append(server.AllowInbound, name)
		for _, address := range staticAddresses(iface.Addresses) {
			prefix, err := netip.ParsePrefix(address)
			if err == nil {
				server.AllowFrom = append(server.AllowFrom, prefix.Masked().String())
			}
		}
	}
	if value.AllowWAN {
		internet := findInternet(cfg)
		if internet == "" {
			return nil, fmt.Errorf("simple dns cannot allow WAN queries without an Internet interface")
		}
		if len(value.WANAllowFrom) == 0 {
			return nil, fmt.Errorf("simple dns WAN access requires at least one allowed source")
		}
		for _, source := range value.WANAllowFrom {
			if _, err := netip.ParsePrefix(source); err != nil {
				address, addressErr := netip.ParseAddr(source)
				if addressErr != nil {
					return nil, fmt.Errorf("simple dns WAN source %s must be an IP address or CIDR", source)
				}
				source = address.String()
			}
			server.AllowFrom = append(server.AllowFrom, source)
		}
		server.Listen = append(server.Listen, "iface("+internet+")")
		server.AllowInbound = append(server.AllowInbound, internet)
	}
	cfg.DNS.Server = server

	return cfg, nil
}

func joinHostPort(host, port string) string {
	if strings.Contains(host, ":") {
		return "[" + host + "]:" + port
	}

	return host + ":" + port
}

func addDefaultRoute(cfg *config.Config, destination, via, dev string) {
	if cfg.Routing == nil {
		cfg.Routing = &config.Routing{}
	}

	cfg.Routing.Static = append(cfg.Routing.Static, config.StaticRoute{
		Destination: destination,
		Via:         via,
		Dev:         dev,
	})
}

func removeDefaultRoutes(cfg *config.Config) {
	if cfg.Routing == nil {
		return
	}

	kept := cfg.Routing.Static[:0]
	for _, route := range cfg.Routing.Static {
		if route.Destination != "0.0.0.0/0" && route.Destination != "::/0" {
			kept = append(kept, route)
		}
	}
	cfg.Routing.Static = kept
}

func hasNetwork(networks []Network, name string) bool {
	for _, network := range networks {
		if network.Name == name {
			return true
		}
	}

	return false
}
