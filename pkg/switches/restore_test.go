package switches

import (
	"reflect"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestRestoreDefinitionUndoesANewSwitch(t *testing.T) {
	before := liveState()
	rolledBack := &config.Switch{
		Model: "mikrotik-crs310-8g-2s", Host: "10.0.0.2", Uplink: "sfp-sfpplus1",
		Ports: map[string]*config.SwitchPortConfig{"ether1": access(30)},
		Bonds: map[string]*config.SwitchBondConfig{"bond2": {Members: []string{"ether2", "ether3"}, Port: trunk(0, 10)}},
	}

	got := RestoreDefinition(nil, rolledBack, &before)
	want := map[string]*config.SwitchPortConfig{
		"ether1": access(10),
		"ether2": {Mode: "trunk", Tagged: []int{20, 10}},
		"ether3": access(20),
	}
	if got == nil || got.Host != "10.0.0.2" || len(got.Bonds) != 0 || !reflect.DeepEqual(got.Ports, want) {
		t.Fatalf("unexpected restore definition %+v ports=%+v", got, got.Ports)
	}

	after := before
	after.Ports = append([]types.SwitchPort{}, before.Ports...)
	for i, port := range after.Ports {
		switch port.Name {
		case "ether1":
			after.Ports[i].PVID = 30
		case "ether2", "ether3":
			after.Ports[i] = types.SwitchPort{Name: port.Name, Type: "ether", Mode: types.PortBondSlave, Bond: "bond2"}
		}
	}

	after.Ports = append(after.Ports, types.SwitchPort{Name: "bond2", Type: "bond", Bridge: "bridge", Mode: types.PortTrunk, PVID: 1, Tagged: []int{10}})
	after.Bonds = append(append([]types.SwitchBond{}, before.Bonds...), types.SwitchBond{Name: "bond2", Members: []string{"ether2", "ether3"}})

	changes, err := Changes(got, rolledBack, after)
	if err != nil {
		t.Fatal(err)
	}

	wantChanges := []types.PortChange{
		{Kind: types.ChangeDeleteBond, Bond: "bond2"},
		{Kind: types.ChangeSetAccess, Ports: []string{"ether1"}, VLAN: 10},
		{Kind: types.ChangeSetTrunk, Ports: []string{"ether2"}, Tagged: []int{10, 20}},
		{Kind: types.ChangeSetAccess, Ports: []string{"ether3"}, VLAN: 20},
	}
	if !reflect.DeepEqual(changes, wantChanges) {
		t.Fatalf("unexpected rollback changes\n got %+v\nwant %+v", changes, wantChanges)
	}
}

func TestRestoreDefinitionKeepsRestoredSettings(t *testing.T) {
	before := liveState()
	restored := &config.Switch{Model: "m", Host: "h", Ports: map[string]*config.SwitchPortConfig{"ether1": access(10)}}
	rolledBack := &config.Switch{Model: "m", Host: "h", Ports: map[string]*config.SwitchPortConfig{"ether1": access(30), "ether3": access(40)}}

	got := RestoreDefinition(restored, rolledBack, &before)
	if got.Ports["ether1"].VLAN != 10 || got.Ports["ether3"].VLAN != 20 || restored.Ports["ether3"] != nil {
		t.Fatalf("unexpected restore %+v, restored mutated: %+v", got.Ports, restored.Ports)
	}

	if RestoreDefinition(nil, rolledBack, nil) != nil {
		t.Fatal("restored a new switch without a before-image")
	}

	if same := RestoreDefinition(restored, rolledBack, nil); same.Ports["ether3"] != nil {
		t.Fatalf("invented a port without a before-image: %+v", same.Ports)
	}
}
