package switches

import (
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
)

func RestoreDefinition(restored, rolledBack *config.Switch, before *types.SwitchState) *config.Switch {
	if restored == nil && (rolledBack == nil || before == nil) {
		return nil
	}

	var out config.Switch
	if restored != nil {
		out = *restored
	} else {
		out = config.Switch{Model: rolledBack.Model, Host: rolledBack.Host, Port: rolledBack.Port, TLSSkipVerify: rolledBack.TLSSkipVerify, Uplink: rolledBack.Uplink}
	}

	out.Ports = copyPorts(out.Ports)
	out.Bonds = copyBonds(out.Bonds)
	if rolledBack == nil || before == nil {
		return &out
	}

	index := indexState(*before)
	for name := range rolledBack.Ports {
		restorePort(&out, name, index)
	}

	for name, bond := range rolledBack.Bonds {
		if _, kept := out.Bonds[name]; kept {
			continue
		}

		if live, existed := index.bonds[name]; existed {
			out.Bonds[name] = &config.SwitchBondConfig{Members: live.Members, Mode: live.Mode, Port: portFromState(index.ports[name])}
			continue
		}

		for _, member := range bond.Members {
			restorePort(&out, member, index)
		}
	}

	return &out
}

func restorePort(out *config.Switch, name string, index liveIndex) {
	if _, kept := out.Ports[name]; kept {
		return
	}

	if port := portFromState(index.ports[name]); port != nil {
		out.Ports[name] = port
	}
}

func portFromState(port types.SwitchPort) *config.SwitchPortConfig {
	switch {
	case port.Bridge == "":
		return nil
	case port.Mode == types.PortTrunk:
		native := 0
		if port.PVID > 1 {
			native = port.PVID
		}

		return &config.SwitchPortConfig{Mode: "trunk", Native: native, Tagged: port.Tagged}
	case port.Mode == types.PortAccess:
		return &config.SwitchPortConfig{Mode: "access", VLAN: port.PVID}
	}

	return nil
}

func copyPorts(in map[string]*config.SwitchPortConfig) map[string]*config.SwitchPortConfig {
	out := make(map[string]*config.SwitchPortConfig, len(in))
	for name, port := range in {
		out[name] = port
	}

	return out
}

func copyBonds(in map[string]*config.SwitchBondConfig) map[string]*config.SwitchBondConfig {
	out := make(map[string]*config.SwitchBondConfig, len(in))
	for name, bond := range in {
		out[name] = bond
	}

	return out
}
