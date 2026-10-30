package flows

import (
	"net/netip"
	"slices"

	"github.com/ChevalRouting/routier/pkg/config"
)

type Settings struct {
	Sampling   int
	Devices    []string
	Collectors []netip.AddrPort
	Source     netip.Addr
	Retention  int64
	MaxFlows   int
}

func (s Settings) Equal(o Settings) bool {
	return s.Sampling == o.Sampling && slices.Equal(s.Devices, o.Devices) && slices.Equal(s.Collectors, o.Collectors) &&
		s.Source == o.Source && s.Retention == o.Retention && s.MaxFlows == o.MaxFlows
}

func SettingsFromConfig(cfg *config.Config) (Settings, bool) {
	if cfg == nil || !cfg.Monitoring.FlowsEnabled() {
		return Settings{}, false
	}

	flows := cfg.Monitoring.Flows
	s := Settings{
		Sampling:  flows.SamplingRate(),
		Devices:   cfg.FlowDevices(),
		Retention: int64(flows.Retention.HoursOrDefault()) * 3600,
		MaxFlows:  flows.Retention.MaxFlowsOrDefault(),
	}

	if source, err := netip.ParseAddr(flows.Source); err == nil {
		s.Source = source
	}

	for _, c := range flows.Collectors {
		if addr, err := netip.ParseAddr(c.Address); err == nil {
			s.Collectors = append(s.Collectors, netip.AddrPortFrom(addr, uint16(c.PortOrDefault())))
		}
	}

	return s, true
}
