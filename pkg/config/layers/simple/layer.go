package simple

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/config/layers"
)

type Layer struct{}

func (Layer) Name() string {
	return "simple"
}

func (Layer) Sections() []string {
	return []string{"configuration", "internet", "networks", "port_forwards", "dns", "system", "hostname", "users", "monitoring"}
}

func (l Layer) Project(cfg *config.Config) (*layers.Projection, error) {
	projected := inspect(cfg)
	sections := map[string]any{
		"internet":      projected.Internet,
		"networks":      projected.Networks,
		"port_forwards": projected.PortForwards,
		"dns":           projected.DNS,
		"system":        projected.System,
		"hostname":      projected.System.Hostname,
		"users":         projected.Users,
		"monitoring":    projected.Monitoring,
	}

	return &layers.Projection{
		Layer: l.Name(), Sections: sections, Issues: projected.Issues, Losses: losses(cfg, projected),
	}, nil
}

func (Layer) ProjectSection(cfg *config.Config, section string) (any, error) {
	projected := inspect(cfg)
	switch section {
	case "configuration":
		return projected, nil
	case "internet":
		return projected.Internet, nil
	case "networks":
		return projected.Networks, nil
	case "port_forwards":
		return projected.PortForwards, nil
	case "dns":
		return projected.DNS, nil
	case "system":
		return projected.System, nil
	case "hostname":
		return projected.System.Hostname, nil
	case "users":
		return projected.Users, nil
	case "monitoring":
		return projected.Monitoring, nil
	default:
		return nil, fmt.Errorf("unknown simple section: %s", section)
	}
}

func (Layer) Build(body json.RawMessage) (*config.Config, error) {
	var value Config
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, fmt.Errorf("invalid simple configuration data: %w", err)
	}

	return build(value)
}

func (Layer) ReplaceSection(cfg *config.Config, section string, body json.RawMessage) (*config.Config, error) {
	projected := inspect(cfg)
	switch section {
	case "configuration":
		if err := json.Unmarshal(body, &projected); err != nil {
			return nil, fmt.Errorf("invalid simple configuration data: %w", err)
		}
	case "internet":
		if err := json.Unmarshal(body, &projected.Internet); err != nil {
			return nil, fmt.Errorf("invalid simple internet data: %w", err)
		}
	case "networks":
		if err := json.Unmarshal(body, &projected.Networks); err != nil {
			return nil, fmt.Errorf("invalid simple networks data: %w", err)
		}
	case "port_forwards":
		if err := json.Unmarshal(body, &projected.PortForwards); err != nil {
			return nil, fmt.Errorf("invalid simple port forwards data: %w", err)
		}
	case "dns":
		if err := json.Unmarshal(body, &projected.DNS); err != nil {
			return nil, fmt.Errorf("invalid simple dns data: %w", err)
		}
	case "system":
		if err := json.Unmarshal(body, &projected.System); err != nil {
			return nil, fmt.Errorf("invalid simple system data: %w", err)
		}
	case "hostname":
		if err := json.Unmarshal(body, &projected.System.Hostname); err != nil {
			return nil, fmt.Errorf("invalid simple hostname data: %w", err)
		}
	case "users":
		if err := json.Unmarshal(body, &projected.Users); err != nil {
			return nil, fmt.Errorf("invalid simple users data: %w", err)
		}
	case "monitoring":
		if err := json.Unmarshal(body, &projected.Monitoring); err != nil {
			return nil, fmt.Errorf("invalid simple monitoring data: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown simple section: %s", section)
	}

	return applyOnto(cfg, projected)
}

func build(value Config) (*config.Config, error) {
	return applyOnto(&config.Config{}, value)
}

func applyOnto(next *config.Config, value Config) (*config.Config, error) {
	next.Version = config.CurrentVersion
	next.Hostname = value.System.Hostname
	next.Users = value.Users
	next.Monitoring = value.Monitoring

	var err error
	next, err = replaceInternet(next, value.Internet)
	if err != nil {
		return nil, err
	}
	next, err = replaceNetworks(next, value.Networks)
	if err != nil {
		return nil, err
	}
	next, err = replacePortForwards(next, value.PortForwards)
	if err != nil {
		return nil, err
	}
	next, err = replaceDNS(next, value.DNS)
	if err != nil {
		return nil, err
	}
	syncFirewallDefaults(next, value.Internet, value.Networks)

	return next, nil
}

func losses(original *config.Config, projected Config) []layers.Loss {
	rebuilt, err := build(projected)
	if err != nil {
		return []layers.Loss{{Path: "configuration", Message: "Simple mode cannot rebuild this configuration: " + err.Error()}}
	}

	checks := []struct {
		path          string
		original      any
		reconstructed any
	}{
		{"interfaces", original.Interfaces, rebuilt.Interfaces},
		{"routing", original.Routing, rebuilt.Routing},
		{"dhcp", original.DHCP, rebuilt.DHCP},
		{"dns", original.DNS, rebuilt.DNS},
		{"nftables", original.Nftables, rebuilt.Nftables},
		{"vrfs", original.VRFs, rebuilt.VRFs},
		{"tunnels", original.Tunnels, rebuilt.Tunnels},
		{"wireguard", original.Wireguard, rebuilt.Wireguard},
		{"sysctl", original.Sysctl, rebuilt.Sysctl},
		{"users", original.Users, rebuilt.Users},
		{"services", original.Services, rebuilt.Services},
		{"logging", original.Logging, rebuilt.Logging},
		{"friends", original.Friends, rebuilt.Friends},
		{"ha", original.HA, rebuilt.HA},
		{"ssh", original.SSH, rebuilt.SSH},
		{"boot_modules", original.BootModules, rebuilt.BootModules},
		{"gai", original.GAI, rebuilt.GAI},
		{"monitoring", original.Monitoring, rebuilt.Monitoring},
	}

	var result []layers.Loss
	for _, check := range checks {
		if !reflect.DeepEqual(check.original, check.reconstructed) {
			result = append(result, layers.Loss{
				Path: check.path, Message: check.path + " contains advanced settings not shown in Simple mode (kept as-is)",
			})
		}
	}

	return result
}
