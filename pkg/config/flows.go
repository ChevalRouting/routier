package config

import (
	"fmt"
	"net"
	"sort"
)

const (
	DefaultFlowSampling     = 1000
	DefaultFlowPort         = 6343
	DefaultFlowRetentionHrs = 72
	DefaultFlowMaxFlows     = 200000
	MaxFlowSampling         = 1 << 20
	FlowLogGroup            = 6343
	FlowSnaplen             = 128
)

func (m *MonitoringConfig) FlowsEnabled() bool {
	return m != nil && m.Flows != nil && m.Flows.Enabled
}

func (f *FlowsConfig) SamplingRate() int {
	if f.Sampling > 0 {
		return f.Sampling
	}

	return DefaultFlowSampling
}

func (c FlowCollector) PortOrDefault() int {
	if c.Port > 0 {
		return c.Port
	}

	return DefaultFlowPort
}

func (r FlowRetention) HoursOrDefault() int {
	if r.Hours > 0 {
		return r.Hours
	}

	return DefaultFlowRetentionHrs
}

func (r FlowRetention) MaxFlowsOrDefault() int {
	if r.MaxFlows > 0 {
		return r.MaxFlows
	}

	return DefaultFlowMaxFlows
}

type deviceSet map[string]bool

func (s deviceSet) add(dev string) {
	if dev != "" {
		s[dev] = true
	}
}

func (s deviceSet) sorted() []string {
	devices := make([]string, 0, len(s))
	for dev := range s {
		devices = append(devices, dev)
	}

	sort.Strings(devices)
	return devices
}

func (cfg *Config) FlowDevices() []string {
	flows := cfg.Monitoring.Flows
	devices := deviceSet{}
	for _, name := range flows.Interfaces {
		if iface := cfg.Interfaces[name]; iface != nil {
			devices.add(iface.Device)
		} else {
			devices.add(name)
		}
	}

	if len(flows.Interfaces) == 0 {
		for _, iface := range cfg.Interfaces {
			devices.add(iface.Device)
		}

		for name := range cfg.VPN.Wireguard {
			devices.add(name)
		}
	}

	return devices.sorted()
}

func validateFlows(cfg *Config, add addfunc) {
	if !cfg.Monitoring.FlowsEnabled() {
		return
	}

	flows := cfg.Monitoring.Flows
	if flows.Sampling < 0 || flows.Sampling > MaxFlowSampling {
		add("monitoring.flows.sampling must be between 1 and %d", MaxFlowSampling)
	}

	for i, name := range flows.Interfaces {
		_, iface := cfg.Interfaces[name]
		_, wg := cfg.VPN.Wireguard[name]
		if !iface && !wg {
			add("monitoring.flows.interfaces[%d]: %q is not a configured interface", i, name)
		}
	}

	if flows.Source != "" && net.ParseIP(flows.Source) == nil {
		add("monitoring.flows.source %q is not a valid IP", flows.Source)
	}

	seen := map[string]bool{}
	for i, c := range flows.Collectors {
		path := fmt.Sprintf("monitoring.flows.collectors[%d]", i)
		if net.ParseIP(c.Address) == nil {
			add("%s.address %q is not a valid IP", path, c.Address)
		}

		if c.Port < 0 || c.Port > 65535 {
			add("%s.port must be between 1 and 65535", path)
		}

		key := net.JoinHostPort(c.Address, fmt.Sprint(c.PortOrDefault()))
		if seen[key] {
			add("%s duplicates collector %s", path, key)
		}

		seen[key] = true
	}

	if flows.Retention.Hours < 0 {
		add("monitoring.flows.retention.hours must be positive")
	}

	if flows.Retention.MaxFlows < 0 {
		add("monitoring.flows.retention.max_flows must be positive")
	}
}
