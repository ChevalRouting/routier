package simple

import (
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/config/layers"
)

type Internet struct {
	Interface string   `json:"interface"`
	Select    string   `json:"select"`
	Mode      string   `json:"mode"`
	IPv6      string   `json:"ipv6"`
	Addresses []string `json:"addresses,omitempty"`
	Gateway   string   `json:"gateway,omitempty"`
	GatewayV6 string   `json:"gateway_v6,omitempty"`
	DNS       []string `json:"dns,omitempty"`
	VIPs      []string `json:"vips,omitempty"`
}

type Network struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Select          string   `json:"select"`
	Addresses       []string `json:"addresses"`
	ManageDHCP      bool     `json:"manage_dhcp"`
	ManageDHCP6     bool     `json:"manage_dhcp6"`
	Pool            string   `json:"pool,omitempty"`
	PoolV6          string   `json:"pool_v6,omitempty"`
	Gateway         string   `json:"gateway,omitempty"`
	DNS             []string `json:"dns,omitempty"`
	Exclusions      []string `json:"exclusions,omitempty"`
	ValidLifetime   int      `json:"valid_lifetime,omitempty"`
	ValidLifetimeV6 int      `json:"valid_lifetime_v6,omitempty"`
	Masquerade      bool     `json:"masquerade"`
	Editable        bool     `json:"editable"`
	Issue           string   `json:"issue,omitempty"`
}

type PortForward struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	Proto    string `json:"proto"`
	Port     string `json:"port"`
	ToHost   string `json:"to_host"`
	ToPort   string `json:"to_port,omitempty"`
	Editable bool   `json:"editable"`
	Issue    string `json:"issue,omitempty"`
}

type DNS struct {
	Enabled      bool             `json:"enabled"`
	Upstreams    []string         `json:"upstreams"`
	Networks     []string         `json:"networks"`
	AllowWAN     bool             `json:"allow_wan"`
	WANAllowFrom []string         `json:"wan_allow_from"`
	DNSSEC       bool             `json:"dnssec"`
	Cache        bool             `json:"cache"`
	Prefetch     bool             `json:"prefetch"`
	ServeExpired bool             `json:"serve_expired"`
	Zones        []config.DNSZone `json:"zones"`
}

type Config struct {
	Internet     *Internet                `json:"internet,omitempty"`
	Networks     []Network                `json:"networks"`
	PortForwards []PortForward            `json:"port_forwards"`
	DNS          DNS                      `json:"dns"`
	System       System                   `json:"system"`
	Users        map[string]*config.User  `json:"users"`
	Monitoring   *config.MonitoringConfig `json:"monitoring,omitempty"`
	Issues       []layers.Issue           `json:"issues,omitempty"`
}

type System struct {
	Hostname string `json:"hostname"`
}
