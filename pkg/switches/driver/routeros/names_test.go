package routeros

import (
	"slices"
	"testing"

	"github.com/ChevalRouting/routier/pkg/types"
)

func renamedSnapshot() snapshot {
	s := snapshot{
		interfaces: []interfaceRecord{
			{ID: "*7", Name: "medias-nfs", DefaultName: "sfp-sfpplus7", Type: "ether", Running: "true"},
			{ID: "*8", Name: "sfp-sfpplus8", DefaultName: "sfp-sfpplus8", Type: "ether"},
			{ID: "*5", Name: "sfp-sfpplus5", DefaultName: "sfp-sfpplus5", Type: "ether"},
			{ID: "*6", Name: "uplink-b", DefaultName: "sfp-sfpplus6", Type: "ether"},
			{ID: "*B", Name: "rt-core-1", Type: "bond"},
		},
		bridges: []bridgeRecord{{ID: "*C", Name: "bridge", VLANFiltering: "true"}},
		ports: []bridgePortRecord{
			{ID: "*1", Interface: "medias-nfs", Bridge: "bridge", PVID: "1", FrameTypes: frameTypesTaggedOnly},
			{ID: "*2", Interface: "sfp-sfpplus8", Bridge: "bridge", PVID: "1", FrameTypes: frameTypesAll},
			{ID: "*3", Interface: "rt-core-1", Bridge: "bridge", PVID: "1", FrameTypes: frameTypesTaggedOnly},
		},
		vlans: []bridgeVLANRecord{
			{ID: "*1", Bridge: "bridge", VLANIDs: "50", Tagged: "bridge,rt-core-1,medias-nfs", Untagged: "sfp-sfpplus8"},
		},
		bonds: []bondingRecord{{ID: "*B", Name: "rt-core-1", Slaves: "sfp-sfpplus5,uplink-b", Mode: "802.3ad"}},
		rates: map[string]string{},
	}
	s.names = namesOf(s.interfaces)
	s.names.normalize(&s)
	return s
}

func findPort(state types.SwitchState, name string) (types.SwitchPort, bool) {
	index := slices.IndexFunc(state.Ports, func(port types.SwitchPort) bool { return port.Name == name })
	if index < 0 {
		return types.SwitchPort{}, false
	}

	return state.Ports[index], true
}

func TestStateUsesDefaultNames(t *testing.T) {
	state := buildState(renamedSnapshot())

	port, ok := findPort(state, "sfp-sfpplus7")
	if !ok {
		t.Fatalf("sfp-sfpplus7 missing from %+v", state.Ports)
	}

	if port.Label != "medias-nfs" || port.Mode != types.PortTrunk || !slices.Equal(port.Tagged, []int{50}) {
		t.Fatalf("unexpected renamed port: %+v", port)
	}

	if _, ok := findPort(state, "medias-nfs"); ok {
		t.Fatal("renamed port reported under its custom name")
	}

	member, _ := findPort(state, "sfp-sfpplus6")
	if member.Mode != types.PortBondSlave || member.Bond != "rt-core-1" {
		t.Fatalf("renamed bond member not resolved: %+v", member)
	}

	plain, _ := findPort(state, "sfp-sfpplus8")
	if plain.Label != "" {
		t.Fatalf("unrenamed port has a label: %+v", plain)
	}
}

func TestPlanUsesDeviceNames(t *testing.T) {
	p := newPlanner(nil, renamedSnapshot())
	p.apply(types.PortChange{Kind: types.ChangeSetAccess, Ports: []string{"sfp-sfpplus7"}, VLAN: 50})
	p.apply(types.PortChange{Kind: types.ChangeCreateBond, Bond: "bond2", Ports: []string{"sfp-sfpplus8", "sfp-sfpplus6"}})

	var got []string
	for _, step := range p.finalize() {
		got = append(got, step.Describe())
	}

	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/vlan/*1 tagged="bridge,rt-core-1" untagged="" vlan-ids="50"`,
		`DELETE /rest/interface/bridge/port/*2`,
		`PUT /rest/interface/bonding mode="802.3ad" name="bond2" slaves="sfp-sfpplus8,uplink-b"`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-all" interface="bond2" pvid="1"`,
		`PATCH /rest/interface/bridge/port/*1 frame-types="admit-only-untagged-and-priority-tagged" pvid="50"`,
		`PATCH /rest/interface/bridge/vlan/*1 tagged="bridge,rt-core-1" untagged="medias-nfs,bond2"`,
	})
}
