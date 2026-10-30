package setup

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/nftables/nat"
)

const (
	WANInterface = "wan"
	firewallTag  = "simple"
	modeDHCP     = "dhcp"
	modeStatic   = "static"
)

type Uplink struct {
	Select  string   `json:"select"`
	Mode    string   `json:"mode" validate:"required,oneof=dhcp static"`
	Address string   `json:"address,omitempty" validate:"optional"`
	Gateway string   `json:"gateway,omitempty" validate:"optional"`
	DNS     []string `json:"dns,omitempty" validate:"optional"`
}

type LAN struct {
	Name    string `json:"name"`
	Select  string `json:"select"`
	Address string `json:"address"`
}

type Network struct {
	Hostname string  `json:"hostname"`
	Uplink   *Uplink `json:"uplink,omitempty" validate:"optional"`
	LANs     []LAN   `json:"lans"`
}

func Apply(cfg *config.Config, network Network) error {
	if strings.TrimSpace(network.Hostname) == "" {
		return fmt.Errorf("a hostname is required")
	}

	cfg.Hostname = strings.TrimSpace(network.Hostname)
	if cfg.Interfaces == nil {
		cfg.Interfaces = map[string]*config.Interface{}
	}

	if err := applyUplink(cfg, network.Uplink); err != nil {
		return err
	}

	for _, lan := range network.LANs {
		if err := applyLAN(cfg, lan); err != nil {
			return err
		}
	}

	masquerade(cfg, network)
	firewallDefaults(cfg, network)
	return nil
}

func applyUplink(cfg *config.Config, uplink *Uplink) error {
	removeDefaultRoutes(cfg)
	if uplink == nil {
		delete(cfg.Interfaces, WANInterface)
		return nil
	}

	if uplink.Select == "" {
		return fmt.Errorf("the uplink needs a port")
	}

	addresses := []string{modeDHCP, "slaac"}
	if uplink.Mode == modeStatic {
		prefix, err := netip.ParsePrefix(uplink.Address)
		if err != nil || !prefix.Addr().Is4() {
			return fmt.Errorf("uplink address %q must be an IPv4 address with its prefix length", uplink.Address)
		}

		gateway, err := netip.ParseAddr(uplink.Gateway)
		if err != nil || !gateway.Is4() {
			return fmt.Errorf("uplink gateway %q must be an IPv4 address", uplink.Gateway)
		}

		addresses = []string{uplink.Address, "slaac"}
		if cfg.Routing == nil {
			cfg.Routing = &config.Routing{}
		}

		cfg.Routing.Static = append(cfg.Routing.Static, config.StaticRoute{Destination: "0.0.0.0/0", Via: uplink.Gateway, Dev: WANInterface})
	}

	cfg.Interfaces[WANInterface] = &config.Interface{Select: uplink.Select, Addresses: addresses}
	if len(uplink.DNS) > 0 {
		if cfg.DNS == nil {
			cfg.DNS = &config.DNS{}
		}

		cfg.DNS.Nameservers = append([]string(nil), uplink.DNS...)
	}

	return nil
}

func applyLAN(cfg *config.Config, lan LAN) error {
	if lan.Name == "" || lan.Select == "" {
		return fmt.Errorf("a local network needs a name and a port")
	}

	if lan.Name == WANInterface {
		return fmt.Errorf("%s is the uplink", WANInterface)
	}

	prefix, err := netip.ParsePrefix(lan.Address)
	if err != nil {
		return fmt.Errorf("local network %s: %q must be an address with its prefix length", lan.Name, lan.Address)
	}

	cfg.Interfaces[lan.Name] = &config.Interface{Select: lan.Select, Addresses: []string{prefix.String()}}
	return nil
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

func nftName(name string) string {
	return strings.NewReplacer(":", "_", ".", "_", "-", "_").Replace(name)
}

func source(name string) string {
	return "$" + nftName(name) + "_network"
}

func masquerade(cfg *config.Config, network Network) {
	owned := map[string]bool{}
	for _, lan := range network.LANs {
		owned[source(lan.Name)] = true
	}

	var kept []nat.Spec
	for _, spec := range nat.Parse(cfg.Nftables) {
		if spec.Kind != nat.KindMasquerade || !owned[spec.Source] {
			kept = append(kept, spec)
		}
	}

	if network.Uplink != nil {
		out := "$" + nftName(WANInterface) + "_interfaces"
		for _, lan := range network.LANs {
			if isIPv4(lan.Address) {
				kept = append(kept, nat.Spec{Kind: nat.KindMasquerade, Source: source(lan.Name), Out: out, Family: "ip"})
			}
		}
	}

	nat.Apply(cfg, kept)
}

func isIPv4(address string) bool {
	prefix, err := netip.ParsePrefix(address)
	return err == nil && prefix.Addr().Is4()
}

func firewallDefaults(cfg *config.Config, network Network) {
	if cfg.Nftables == nil {
		cfg.Nftables = &config.NftablesConfig{}
	}

	if cfg.Nftables.Chains == nil {
		cfg.Nftables.Chains = map[string]*config.NftChain{}
	}

	replaceTagged(cfg, "input", []config.ManagedRule{
		{Tag: firewallTag, Comment: "allow loopback", Match: &config.RuleMatch{IIF: "lo"}, Action: "accept"},
		{Tag: firewallTag, Comment: "allow established traffic", Match: &config.RuleMatch{CTState: "established,related"}, Action: "accept"},
		{Tag: firewallTag, Comment: "drop invalid traffic", Match: &config.RuleMatch{CTState: "invalid"}, Action: "drop"},
		{Tag: firewallTag, Comment: "allow IPv4 ICMP", Match: &config.RuleMatch{Protocol: "icmp"}, Action: "accept"},
		{Tag: firewallTag, Comment: "allow IPv6 neighbor discovery", Match: &config.RuleMatch{Protocol: "icmpv6"}, Action: "accept"},
		{Tag: firewallTag, Comment: "allow administration", Match: &config.RuleMatch{Protocol: "tcp", DPort: "ssh, 8080"}, Action: "accept"},
	})

	forward := []config.ManagedRule{
		{Tag: firewallTag, Comment: "allow established forwarding", Match: &config.RuleMatch{CTState: "established,related"}, Action: "accept"},
		{Tag: firewallTag, Comment: "drop invalid forwarding", Match: &config.RuleMatch{CTState: "invalid"}, Action: "drop"},
	}
	if network.Uplink != nil {
		out := "$" + nftName(WANInterface) + "_interfaces"
		for _, lan := range network.LANs {
			if isIPv4(lan.Address) {
				forward = append(forward, config.ManagedRule{
					Tag: firewallTag, Comment: "allow " + lan.Name + " to Internet",
					Match: &config.RuleMatch{IIF: "$" + nftName(lan.Name) + "_interfaces", OIF: out}, Action: "accept",
				})
			}
		}
	}

	replaceTagged(cfg, "forward", forward)
}

func replaceTagged(cfg *config.Config, chain string, generated []config.ManagedRule) {
	current := cfg.Nftables.Chains[chain]
	if current == nil {
		current = &config.NftChain{}
		cfg.Nftables.Chains[chain] = current
	}

	kept := current.Managed[:0]
	for _, rule := range current.Managed {
		if rule.Tag != firewallTag {
			kept = append(kept, rule)
		}
	}

	current.Managed = append(kept, generated...)
}
