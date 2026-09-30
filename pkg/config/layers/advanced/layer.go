package advanced

import (
	"encoding/json"
	"fmt"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/config/layers"
)

type Layer struct{}

func (Layer) Name() string {
	return "advanced"
}

func (Layer) Sections() []string {
	return []string{
		"interfaces", "tunnels", "routing", "wireguard", "nftables", "sysctl",
		"dns", "dns_server", "users", "services", "hostname", "logging",
		"ha", "ssh", "vrfs", "monitoring", "dhcp",
	}
}

func (l Layer) Project(cfg *config.Config) (*layers.Projection, error) {
	sections := make(map[string]any, len(l.Sections()))
	for _, name := range l.Sections() {
		section, err := l.ProjectSection(cfg, name)
		if err != nil {
			return nil, err
		}

		sections[name] = section
	}

	return &layers.Projection{Layer: l.Name(), Sections: sections}, nil
}

func (Layer) ProjectSection(cfg *config.Config, section string) (any, error) {
	switch section {
	case "interfaces":
		return cfg.Interfaces, nil
	case "tunnels":
		return cfg.Tunnels, nil
	case "routing":
		return cfg.Routing, nil
	case "wireguard":
		return cfg.Wireguard, nil
	case "nftables":
		return cfg.Nftables, nil
	case "sysctl":
		return cfg.Sysctl, nil
	case "dns":
		return cfg.DNS, nil
	case "dns_server":
		if cfg.DNS == nil {
			return nil, nil
		}

		return cfg.DNS.Server, nil
	case "users":
		return cfg.Users, nil
	case "services":
		return cfg.Services, nil
	case "hostname":
		return cfg.Hostname, nil
	case "logging":
		return cfg.Logging, nil
	case "ha":
		return cfg.HA, nil
	case "ssh":
		return cfg.SSH, nil
	case "vrfs":
		return cfg.VRFs, nil
	case "monitoring":
		return cfg.Monitoring, nil
	case "dhcp":
		return cfg.DHCP, nil
	default:
		return nil, fmt.Errorf("unknown advanced section: %s", section)
	}
}

func (Layer) ReplaceSection(_ *config.Config, section string, _ json.RawMessage) (*config.Config, error) {
	return nil, fmt.Errorf("advanced section %s uses the canonical section editor", section)
}

func (Layer) Build(body json.RawMessage) (*config.Config, error) {
	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("invalid advanced configuration data: %w", err)
	}

	return &cfg, nil
}
