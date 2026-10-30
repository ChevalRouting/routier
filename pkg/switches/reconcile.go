package switches

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
)

const defaultBondMode = "802.3ad"

var ErrVLANFilteringOff = errors.New("bridge VLAN filtering is off on the switch")

type liveIndex struct {
	ports map[string]types.SwitchPort
	bonds map[string]types.SwitchBond
}

func indexState(state types.SwitchState) liveIndex {
	index := liveIndex{ports: make(map[string]types.SwitchPort), bonds: make(map[string]types.SwitchBond)}
	for _, port := range state.Ports {
		index.ports[port.Name] = port
	}

	for _, bond := range state.Bonds {
		index.bonds[bond.Name] = bond
	}

	return index
}

func Changes(desired, previous *config.Switch, live types.SwitchState) ([]types.PortChange, error) {
	if !live.VLANFiltering {
		return nil, ErrVLANFilteringOff
	}

	index := indexState(live)
	var changes []types.PortChange
	bondChanges, err := bondChanges(desired, previous, index)
	if err != nil {
		return nil, err
	}

	changes = append(changes, bondChanges...)
	portChanges, err := portChanges(desired, index, deletedBonds(bondChanges))
	if err != nil {
		return nil, err
	}

	changes = append(changes, portChanges...)
	return changes, nil
}

func bondChanges(desired, previous *config.Switch, index liveIndex) ([]types.PortChange, error) {
	released := releasedBonds(desired, previous, index)
	changes := make([]types.PortChange, 0, len(released))
	for _, name := range sortedNames(released) {
		changes = append(changes, types.PortChange{Kind: types.ChangeDeleteBond, Bond: name})
	}

	for _, name := range sortedNames(desired.Bonds) {
		bond := desired.Bonds[name]
		if _, exists := index.bonds[name]; exists && !released[name] {
			if bond.Port != nil && portDiffers(index.ports[name], bond.Port) {
				changes = append(changes, portChange(name, bond.Port))
			}

			continue
		}

		if err := checkMembers(name, bond.Members, index, released); err != nil {
			return nil, err
		}

		changes = append(changes, types.PortChange{Kind: types.ChangeCreateBond, Bond: name, Ports: slices.Clone(bond.Members), Mode: bondMode(bond)})
		if bond.Port != nil {
			changes = append(changes, portChange(name, bond.Port))
		}
	}

	return changes, nil
}

func releasedBonds(desired, previous *config.Switch, index liveIndex) map[string]bool {
	released := map[string]bool{}
	if previous != nil {
		for name := range previous.Bonds {
			_, kept := desired.Bonds[name]
			if _, exists := index.bonds[name]; exists && !kept {
				released[name] = true
			}
		}
	}

	for name, bond := range desired.Bonds {
		if live, exists := index.bonds[name]; exists && (!sameMembers(live.Members, bond.Members) || bondMode(bond) != live.Mode) {
			released[name] = true
		}
	}

	for _, member := range claimedPorts(desired) {
		port := index.ports[member]
		if port.Mode != types.PortBondSlave {
			continue
		}

		if _, kept := desired.Bonds[port.Bond]; kept && !released[port.Bond] {
			continue
		}

		if _, exists := index.bonds[port.Bond]; exists {
			released[port.Bond] = true
		}
	}

	return released
}

func bondMode(bond *config.SwitchBondConfig) string {
	if bond.Mode == "" {
		return defaultBondMode
	}

	return bond.Mode
}

func claimedPorts(desired *config.Switch) []string {
	claimed := sortedNames(desired.Ports)
	for _, name := range sortedNames(desired.Bonds) {
		claimed = append(claimed, desired.Bonds[name].Members...)
	}

	return claimed
}

func checkMembers(bond string, members []string, index liveIndex, released map[string]bool) error {
	for _, member := range members {
		port, ok := index.ports[member]
		if !ok {
			return fmt.Errorf("bond %s: port %s not found on the switch", bond, member)
		}

		if port.Mode == types.PortBondSlave && !released[port.Bond] {
			return fmt.Errorf("bond %s: port %s already belongs to %s", bond, member, port.Bond)
		}
	}

	return nil
}

func deletedBonds(changes []types.PortChange) map[string]bool {
	deleted := map[string]bool{}
	for _, change := range changes {
		if change.Kind == types.ChangeDeleteBond {
			deleted[change.Bond] = true
		}
	}

	return deleted
}

func portChanges(desired *config.Switch, index liveIndex, deleted map[string]bool) ([]types.PortChange, error) {
	var changes []types.PortChange
	for _, name := range sortedNames(desired.Ports) {
		live, ok := index.ports[name]
		if !ok {
			return nil, fmt.Errorf("port %s not found on the switch", name)
		}

		if live.Mode == types.PortBondSlave && !deleted[live.Bond] {
			return nil, fmt.Errorf("port %s belongs to bond %s, configure the bond instead", name, live.Bond)
		}

		if live.Mode == types.PortBondSlave || portDiffers(live, desired.Ports[name]) {
			changes = append(changes, portChange(name, desired.Ports[name]))
		}
	}

	return changes, nil
}

func portChange(name string, want *config.SwitchPortConfig) types.PortChange {
	if want.Mode == "access" {
		return types.PortChange{Kind: types.ChangeSetAccess, Ports: []string{name}, VLAN: want.VLAN}
	}

	return types.PortChange{Kind: types.ChangeSetTrunk, Ports: []string{name}, VLAN: want.Native, Tagged: trunkTagged(want)}
}

func trunkTagged(want *config.SwitchPortConfig) []int {
	var tagged []int
	for _, vid := range want.Tagged {
		if vid != want.Native && !slices.Contains(tagged, vid) {
			tagged = append(tagged, vid)
		}
	}

	sort.Ints(tagged)
	return tagged
}

func portDiffers(live types.SwitchPort, want *config.SwitchPortConfig) bool {
	if live.Bridge == "" {
		return true
	}

	if want.Mode == "access" {
		return live.Mode != types.PortAccess || live.PVID != want.VLAN || len(live.Tagged) > 0
	}

	pvid := max(want.Native, 1)
	tagged := slices.Clone(live.Tagged)
	sort.Ints(tagged)
	return live.Mode != types.PortTrunk || live.PVID != pvid || !slices.Equal(tagged, trunkTagged(want))
}

func sameMembers(a, b []string) bool {
	left, right := slices.Clone(a), slices.Clone(b)
	sort.Strings(left)
	sort.Strings(right)
	return slices.Equal(left, right)
}

func sortedNames[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}
