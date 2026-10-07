package configlayerstest

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/config/layers/simple"
)

func TestSimpleProjectionRecognizesInternetAndNetwork(t *testing.T) {
	cfg := fixture()
	layer := simple.Layer{}
	projection, err := layer.Project(cfg)
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	internet, ok := projection.Sections["internet"].(*simple.Internet)
	if !ok || internet == nil {
		t.Fatalf("internet projection = %#v", projection.Sections["internet"])
	}

	if internet.Interface != "wan" || internet.Mode != "dhcp" {
		t.Fatalf("internet = %#v", internet)
	}

	networks, ok := projection.Sections["networks"].([]simple.Network)
	if !ok || len(networks) != 1 {
		t.Fatalf("networks = %#v", projection.Sections["networks"])
	}

	if networks[0].Name != "lan" || !networks[0].ManageDHCP || networks[0].Pool != "192.168.1.100-192.168.1.250" {
		t.Fatalf("network = %#v", networks[0])
	}
}

func TestSimpleNetworkSaveRebuildsCanonicalConfig(t *testing.T) {
	cfg := fixture()
	cfg.Sysctl = map[string]string{"net.ipv4.ip_forward": "1"}
	layer := simple.Layer{}
	section, err := layer.ProjectSection(cfg, "networks")
	if err != nil {
		t.Fatalf("project networks: %v", err)
	}

	body, err := json.Marshal(section)
	if err != nil {
		t.Fatalf("marshal networks: %v", err)
	}

	got, err := layer.ReplaceSection(cfg, "networks", body)
	if err != nil {
		t.Fatalf("replace networks: %v", err)
	}

	if got.Sysctl != nil {
		t.Fatalf("simple rebuild preserved unsupported sysctl: %#v", got.Sysctl)
	}

	if got.Interfaces["lan"] == nil || got.DHCP == nil || len(got.DHCP.Subnets4) != 1 {
		t.Fatalf("simple rebuild lost represented network: %#v", got)
	}

	if got.Interfaces["lan"].MTU != 0 {
		t.Fatalf("simple rebuild preserved hidden MTU: %d", got.Interfaces["lan"].MTU)
	}
}

func TestSimpleProjectionReportsLosses(t *testing.T) {
	cfg := fixture()
	cfg.Sysctl = map[string]string{"net.ipv4.ip_forward": "1"}
	layer := simple.Layer{}

	projection, err := layer.Project(cfg)
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	paths := map[string]bool{}
	for _, loss := range projection.Losses {
		paths[loss.Path] = true
	}

	if !paths["interfaces"] || !paths["sysctl"] {
		t.Fatalf("losses = %#v", projection.Losses)
	}
}

func TestSimpleBuildCreatesCanonicalConfig(t *testing.T) {
	layer := simple.Layer{}
	value := simple.Config{
		Internet: &simple.Internet{Interface: "wan", Select: "name=eth0", Mode: "dhcp", IPv6: "slaac"},
		Networks: []simple.Network{{
			Name: "lan", Select: "name=eth1", Addresses: []string{"192.168.1.1/24"},
		}},
		System: simple.System{Hostname: "router"},
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := layer.Build(body)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if got.Version != config.CurrentVersion || got.Hostname != "router" {
		t.Fatalf("canonical identity = %#v", got)
	}

	if !reflect.DeepEqual(got.Interfaces["wan"].Addresses, []string{"dhcp", "slaac"}) {
		t.Fatalf("wan addresses = %#v", got.Interfaces["wan"].Addresses)
	}
}

func TestSimpleProjectionLocksAdvancedNetwork(t *testing.T) {
	cfg := fixture()
	cfg.Interfaces["lan"].VRF = "blue"
	layer := simple.Layer{}
	projection, err := layer.Project(cfg)
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	networks := projection.Sections["networks"].([]simple.Network)
	if len(networks) != 1 || networks[0].Editable {
		t.Fatalf("networks = %#v", networks)
	}

	if len(projection.Issues) != 1 {
		t.Fatalf("issues = %#v", projection.Issues)
	}
}

func TestSimpleNetworkRenameMovesInterfaceAndDHCP(t *testing.T) {
	cfg := fixture()
	layer := simple.Layer{}
	section, err := layer.ProjectSection(cfg, "networks")
	if err != nil {
		t.Fatalf("project networks: %v", err)
	}

	networks := section.([]simple.Network)
	networks[0].Name = "home"
	body, err := json.Marshal(networks)
	if err != nil {
		t.Fatalf("marshal networks: %v", err)
	}

	got, err := layer.ReplaceSection(cfg, "networks", body)
	if err != nil {
		t.Fatalf("replace networks: %v", err)
	}

	if got.Interfaces["lan"] != nil || got.Interfaces["home"] == nil {
		t.Fatalf("interfaces = %#v", got.Interfaces)
	}

	if got.DHCP.Subnets4[0].Interface != "home" {
		t.Fatalf("DHCP interface = %q", got.DHCP.Subnets4[0].Interface)
	}

	if got.DHCP.Subnets4[0].ValidLifetime != 7200 {
		t.Fatalf("valid lifetime = %d", got.DHCP.Subnets4[0].ValidLifetime)
	}
}

func TestSimpleNATMasqueradeAndPortForward(t *testing.T) {
	cfg := fixture()
	layer := simple.Layer{}

	raw, err := layer.ProjectSection(cfg, "configuration")
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	conf := raw.(simple.Config)

	for i := range conf.Networks {
		if conf.Networks[i].Name == "lan" {
			conf.Networks[i].Masquerade = true
		}
	}

	conf.PortForwards = append(conf.PortForwards, simple.PortForward{
		Name: "web", Proto: "tcp", Port: "443", ToHost: "192.168.1.10", ToPort: "8443", Editable: true,
	})

	body, err := json.Marshal(conf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := layer.ReplaceSection(cfg, "configuration", body)
	if err != nil {
		t.Fatalf("replace: %v", err)
	}

	raw, err = layer.ProjectSection(got, "configuration")
	if err != nil {
		t.Fatalf("re-project: %v", err)
	}

	back := raw.(simple.Config)

	var lan *simple.Network
	for i := range back.Networks {
		if back.Networks[i].Name == "lan" {
			lan = &back.Networks[i]
		}
	}

	if lan == nil || !lan.Masquerade {
		t.Fatalf("lan masquerade not projected back: %#v", back.Networks)
	}

	if len(back.PortForwards) != 1 {
		t.Fatalf("port forwards = %#v", back.PortForwards)
	}

	forward := back.PortForwards[0]
	if forward.Proto != "tcp" || forward.Port != "443" || forward.ToHost != "192.168.1.10" || forward.ToPort != "8443" || !forward.Editable {
		t.Fatalf("port forward = %#v", forward)
	}
}

func TestSimpleBuildAddsBaselineFirewallRules(t *testing.T) {
	layer := simple.Layer{}
	value := simple.Config{
		Internet: &simple.Internet{Interface: "wan", Select: "name=eth0", Mode: "dhcp"},
		Networks: []simple.Network{{
			Name: "lan", Select: "name=eth1", Addresses: []string{"192.168.1.1/24"}, Masquerade: true,
		}},
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	cfg, err := layer.Build(body)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	input := cfg.Nftables.Chains["input"].Managed
	for _, want := range []config.ManagedRule{
		{Match: &config.RuleMatch{IIF: "lo"}, Action: "accept"},
		{Match: &config.RuleMatch{CTState: "established,related"}, Action: "accept"},
		{Match: &config.RuleMatch{CTState: "invalid"}, Action: "drop"},
		{Match: &config.RuleMatch{Protocol: "icmp"}, Action: "accept"},
		{Match: &config.RuleMatch{Protocol: "icmpv6"}, Action: "accept"},
		{Match: &config.RuleMatch{Protocol: "tcp", DPort: "ssh, 8080"}, Action: "accept"},
	} {
		if !hasManagedRule(input, want) {
			t.Fatalf("input rules missing %#v: %#v", want, input)
		}
	}

	forward := cfg.Nftables.Chains["forward"].Managed
	if !hasManagedRule(forward, config.ManagedRule{Match: &config.RuleMatch{CTState: "established,related"}, Action: "accept"}) ||
		!hasManagedRule(forward, config.ManagedRule{Match: &config.RuleMatch{CTState: "invalid"}, Action: "drop"}) {
		t.Fatalf("forward rules missing connection tracking baseline: %#v", forward)
	}

	want := config.ManagedRule{
		Match: &config.RuleMatch{IIF: "$lan_interfaces", OIF: "$wan_interfaces"}, Action: "accept",
	}
	if !hasManagedRule(forward, want) {
		t.Fatalf("forward rules missing LAN to WAN allow: %#v", forward)
	}

	value.Networks[0].Masquerade = false
	body, _ = json.Marshal(value)
	cfg, err = layer.Build(body)
	if err != nil {
		t.Fatalf("build without masquerade: %v", err)
	}

	if hasManagedRule(cfg.Nftables.Chains["forward"].Managed, want) {
		t.Fatalf("LAN to WAN allow present without masquerade: %#v", cfg.Nftables.Chains["forward"].Managed)
	}
}

func TestSimpleUsersCanBeProjectedAndReplaced(t *testing.T) {
	layer := simple.Layer{}
	cfg := fixture()
	cfg.Users = map[string]*config.User{
		"routier": {Shell: "/bin/ash", Groups: []string{"wheel"}, PasswordHash: "existing-hash"},
	}

	projected, err := layer.ProjectSection(cfg, "users")
	if err != nil {
		t.Fatalf("project users: %v", err)
	}

	users := projected.(map[string]*config.User)
	users["operator"] = &config.User{Shell: "/bin/ash", Groups: []string{"wheel"}, SSHKeys: []string{"ssh-ed25519 test"}}
	body, err := json.Marshal(users)
	if err != nil {
		t.Fatalf("marshal users: %v", err)
	}

	got, err := layer.ReplaceSection(cfg, "users", body)
	if err != nil {
		t.Fatalf("replace users: %v", err)
	}

	if got.Users["operator"] == nil || got.Users["operator"].SSHKeys[0] != "ssh-ed25519 test" {
		t.Fatalf("operator user not saved: %#v", got.Users)
	}

	if got.Users["routier"] == nil || got.Users["routier"].PasswordHash != "existing-hash" {
		t.Fatalf("existing user password hash was not preserved: %#v", got.Users)
	}
}

func TestSimpleHostnameAliasCanBeProjectedAndReplaced(t *testing.T) {
	layer := simple.Layer{}
	cfg := fixture()

	projected, err := layer.ProjectSection(cfg, "hostname")
	if err != nil {
		t.Fatalf("project hostname: %v", err)
	}

	if projected != "router" {
		t.Fatalf("hostname projection = %#v", projected)
	}

	got, err := layer.ReplaceSection(cfg, "hostname", json.RawMessage(`"edge-router"`))
	if err != nil {
		t.Fatalf("replace hostname: %v", err)
	}

	if got.Hostname != "edge-router" {
		t.Fatalf("hostname = %q", got.Hostname)
	}
}

func hasManagedRule(rules []config.ManagedRule, want config.ManagedRule) bool {
	for _, rule := range rules {
		if rule.Action == want.Action && reflect.DeepEqual(rule.Match, want.Match) {
			return true
		}
	}

	return false
}

func TestSimplePortForwardSupportsTCPAndUDP(t *testing.T) {
	layer := simple.Layer{}
	value := simple.Config{
		Internet: &simple.Internet{Interface: "wan", Select: "name=eth0", Mode: "dhcp"},
		PortForwards: []simple.PortForward{{
			Name: "dns", Proto: "tcp+udp", Port: "53", ToHost: "192.168.1.53", Editable: true,
		}},
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	cfg, err := layer.Build(body)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	projected, err := layer.ProjectSection(cfg, "port_forwards")
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	forwards := projected.([]simple.PortForward)
	if len(forwards) != 1 || forwards[0].Proto != "tcp+udp" {
		t.Fatalf("port forwards = %#v", forwards)
	}
}

func TestSimpleDNSBuildsForwarderForSelectedNetworks(t *testing.T) {
	layer := simple.Layer{}
	value := simple.Config{
		Networks: []simple.Network{{
			Name: "lan", Select: "name=eth1", Addresses: []string{"192.168.1.1/24", "2001:db8:1::1/64"},
		}},
		DNS: simple.DNS{
			Enabled: true, Upstreams: []string{"1.1.1.1"}, Networks: []string{"lan"},
			WANAllowFrom: []string{}, DNSSEC: true, Cache: true, Prefetch: true, ServeExpired: true,
		},
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	cfg, err := layer.Build(body)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	server := cfg.DNS.Server
	if server == nil || !server.Enabled || server.Mode != config.DNSModeForwarder {
		t.Fatalf("dns server = %#v", server)
	}

	if !reflect.DeepEqual(server.Listen, []string{"127.0.0.1", "::1", "iface(lan)"}) {
		t.Fatalf("listen = %#v", server.Listen)
	}

	if !reflect.DeepEqual(server.AllowFrom, []string{"127.0.0.0/8", "::1/128", "192.168.1.0/24", "2001:db8:1::/64"}) {
		t.Fatalf("allow from = %#v", server.AllowFrom)
	}

	projected, err := layer.ProjectSection(cfg, "dns")
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	got := projected.(simple.DNS)
	if !reflect.DeepEqual(got, value.DNS) {
		t.Fatalf("dns projection = %#v", got)
	}
}

func TestSimpleDNSWANAccessUsesExplicitAllowlist(t *testing.T) {
	layer := simple.Layer{}
	value := simple.Config{
		Internet: &simple.Internet{Interface: "wan", Select: "name=eth0", Mode: "dhcp", IPv6: "disabled"},
		DNS: simple.DNS{
			Enabled: true, Upstreams: []string{"9.9.9.9"}, AllowWAN: true,
			WANAllowFrom: []string{"203.0.113.10/32", "2001:db8::/48"}, Cache: true,
		},
	}
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	cfg, err := layer.Build(body)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	server := cfg.DNS.Server
	if !reflect.DeepEqual(server.Listen, []string{"127.0.0.1", "::1", "iface(wan)"}) {
		t.Fatalf("listen = %#v", server.Listen)
	}

	if !reflect.DeepEqual(server.AllowInbound, []string{"wan"}) {
		t.Fatalf("allow inbound = %#v", server.AllowInbound)
	}

	if !reflect.DeepEqual(server.AllowFrom, []string{"127.0.0.0/8", "::1/128", "203.0.113.10/32", "2001:db8::/48"}) {
		t.Fatalf("allow from = %#v", server.AllowFrom)
	}

	projected, err := layer.ProjectSection(cfg, "dns")
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	got := projected.(simple.DNS)
	if !got.AllowWAN || !reflect.DeepEqual(got.WANAllowFrom, value.DNS.WANAllowFrom) {
		t.Fatalf("dns projection = %#v", got)
	}
}

func fixture() *config.Config {
	return &config.Config{
		Version:  config.CurrentVersion,
		Hostname: "router",
		Interfaces: map[string]*config.Interface{
			"wan": {Select: "mac(00:11:22:33:44:55)", Addresses: []string{"dhcp"}, MTU: 1492},
			"lan": {Select: "mac(00:11:22:33:44:66)", Addresses: []string{"192.168.1.1/24"}, MTU: 1500},
		},
		DHCP: &config.DHCP{
			Enabled: true,
			Subnets4: []config.KeaSubnet{
				{
					Subnet:        "192.168.1.0/24",
					Interface:     "lan",
					Pools:         []string{"192.168.1.100-192.168.1.250"},
					Gateway:       "192.168.1.1",
					DNS:           []string{"192.168.1.1"},
					ValidLifetime: 7200,
				},
			},
		},
		DNS: &config.DNS{Nameservers: []string{"1.1.1.1"}},
	}
}
