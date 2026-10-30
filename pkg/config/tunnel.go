package config

import "fmt"

func (iface *Interface) Tunnel() *Tunnel {
	return &Tunnel{Mode: iface.Mode, Local: iface.Local, Remote: iface.Remote, TTL: iface.TTL, Addresses: iface.Addresses, MTU: iface.MTU, Friend: iface.Friend}
}

func (t *Tunnel) Interface() *Interface {
	return &Interface{Type: "tunnel", Mode: t.Mode, Local: t.Local, Remote: t.Remote, TTL: t.TTL, Addresses: t.Addresses, MTU: t.MTU, Friend: t.Friend}
}

func (cfg *Config) Tunnels() map[string]*Tunnel {
	tunnels := map[string]*Tunnel{}
	for name, iface := range cfg.Interfaces {
		if iface != nil && iface.Type == "tunnel" {
			tunnels[name] = iface.Tunnel()
		}
	}

	return tunnels
}

func (cfg *Config) ReplaceTunnels(tunnels map[string]*Tunnel) error {
	for name, tunnel := range tunnels {
		if tunnel == nil {
			return fmt.Errorf("tunnels.%s must be a mapping", name)
		}

		if existing := cfg.Interfaces[name]; existing != nil && existing.Type != "tunnel" {
			return fmt.Errorf("interfaces.%s is already configured with type %q", name, existing.Type)
		}
	}

	if cfg.Interfaces == nil {
		cfg.Interfaces = map[string]*Interface{}
	}

	for name, iface := range cfg.Interfaces {
		if iface.Type == "tunnel" && tunnels[name] == nil {
			delete(cfg.Interfaces, name)
		}
	}

	for name, tunnel := range tunnels {
		iface := tunnel.Interface()
		if existing := cfg.Interfaces[name]; existing != nil {
			iface.VRF = existing.VRF
			iface.Device = existing.Device
			iface.DHCPOptions = existing.DHCPOptions
		}

		cfg.Interfaces[name] = iface
	}

	return nil
}
