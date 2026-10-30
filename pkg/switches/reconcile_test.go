package switches

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
)

func liveState() types.SwitchState {
	return types.SwitchState{
		VLANFiltering: true,
		Ports: []types.SwitchPort{
			{Name: "ether1", Type: "ether", Bridge: "bridge", Mode: types.PortAccess, PVID: 10},
			{Name: "ether2", Type: "ether", Bridge: "bridge", Mode: types.PortTrunk, PVID: 1, Tagged: []int{20, 10}},
			{Name: "ether3", Type: "ether", Bridge: "bridge", Mode: types.PortAccess, PVID: 20},
			{Name: "ether4", Type: "ether", Mode: types.PortUnbridged},
			{Name: "ether7", Type: "ether", Mode: types.PortBondSlave, Bond: "bond1"},
			{Name: "ether8", Type: "ether", Mode: types.PortBondSlave, Bond: "bond1"},
			{Name: "sfp-sfpplus1", Type: "ether", Bridge: "bridge", Mode: types.PortTrunk, PVID: 1, Tagged: []int{10, 20}},
			{Name: "bond1", Type: "bond", Bridge: "bridge", Mode: types.PortTrunk, PVID: 1, Tagged: []int{20}},
		},
		Bonds: []types.SwitchBond{{Name: "bond1", Members: []string{"ether7", "ether8"}, Mode: "802.3ad"}},
	}
}

func access(vid int) *config.SwitchPortConfig {
	return &config.SwitchPortConfig{Mode: "access", VLAN: vid}
}

func trunk(native int, tagged ...int) *config.SwitchPortConfig {
	return &config.SwitchPortConfig{Mode: "trunk", Native: native, Tagged: tagged}
}

type reconcileCase struct {
	name     string
	desired  *config.Switch
	previous *config.Switch
	want     []types.PortChange
	err      string
}

func TestChanges(t *testing.T) {
	cases := []reconcileCase{
		{
			name:    "already in sync",
			desired: &config.Switch{Ports: map[string]*config.SwitchPortConfig{"ether1": access(10), "ether2": trunk(0, 10, 20)}},
		},
		{
			name:    "access vlan and unbridged port",
			desired: &config.Switch{Ports: map[string]*config.SwitchPortConfig{"ether1": access(30), "ether4": access(10)}},
			want: []types.PortChange{
				{Kind: types.ChangeSetAccess, Ports: []string{"ether1"}, VLAN: 30},
				{Kind: types.ChangeSetAccess, Ports: []string{"ether4"}, VLAN: 10},
			},
		},
		{
			name:    "trunk gains a native vlan and drops duplicates",
			desired: &config.Switch{Ports: map[string]*config.SwitchPortConfig{"ether2": trunk(10, 20, 10, 20)}},
			want:    []types.PortChange{{Kind: types.ChangeSetTrunk, Ports: []string{"ether2"}, VLAN: 10, Tagged: []int{20}}},
		},
		{
			name:    "new bond then its port",
			desired: &config.Switch{Bonds: map[string]*config.SwitchBondConfig{"bond2": {Members: []string{"ether3", "ether4"}, Port: trunk(0, 10)}}},
			want: []types.PortChange{
				{Kind: types.ChangeCreateBond, Bond: "bond2", Ports: []string{"ether3", "ether4"}, Mode: "802.3ad"},
				{Kind: types.ChangeSetTrunk, Ports: []string{"bond2"}, Tagged: []int{10}},
			},
		},
		{
			name:     "bond removed from the config is deleted",
			desired:  &config.Switch{},
			previous: &config.Switch{Bonds: map[string]*config.SwitchBondConfig{"bond1": {Members: []string{"ether7", "ether8"}}}},
			want:     []types.PortChange{{Kind: types.ChangeDeleteBond, Bond: "bond1"}},
		},
		{
			name:    "bond never managed is left alone",
			desired: &config.Switch{},
		},
		{
			name:    "existing bond port changes",
			desired: &config.Switch{Bonds: map[string]*config.SwitchBondConfig{"bond1": {Members: []string{"ether8", "ether7"}, Port: trunk(0, 20, 30)}}},
			want:    []types.PortChange{{Kind: types.ChangeSetTrunk, Ports: []string{"bond1"}, Tagged: []int{20, 30}}},
		},
		{
			name:    "bond members change by recreating the bond",
			desired: &config.Switch{Bonds: map[string]*config.SwitchBondConfig{"bond1": {Members: []string{"ether7", "ether3"}, Port: trunk(0, 20)}}},
			want: []types.PortChange{
				{Kind: types.ChangeDeleteBond, Bond: "bond1"},
				{Kind: types.ChangeCreateBond, Bond: "bond1", Ports: []string{"ether7", "ether3"}, Mode: "802.3ad"},
				{Kind: types.ChangeSetTrunk, Ports: []string{"bond1"}, Tagged: []int{20}},
			},
		},
		{
			name:    "bond mode change recreates the bond",
			desired: &config.Switch{Bonds: map[string]*config.SwitchBondConfig{"bond1": {Members: []string{"ether7", "ether8"}, Mode: "active-backup"}}},
			want: []types.PortChange{
				{Kind: types.ChangeDeleteBond, Bond: "bond1"},
				{Kind: types.ChangeCreateBond, Bond: "bond1", Ports: []string{"ether7", "ether8"}, Mode: "active-backup"},
			},
		},
		{
			name:    "bond member configured as a port releases its bond",
			desired: &config.Switch{Ports: map[string]*config.SwitchPortConfig{"ether7": access(10)}},
			want: []types.PortChange{
				{Kind: types.ChangeDeleteBond, Bond: "bond1"},
				{Kind: types.ChangeSetAccess, Ports: []string{"ether7"}, VLAN: 10},
			},
		},
		{
			name:     "removed bond members join a new bond",
			desired:  &config.Switch{Bonds: map[string]*config.SwitchBondConfig{"bond2": {Members: []string{"ether8", "ether3"}, Mode: "active-backup"}}},
			previous: &config.Switch{Bonds: map[string]*config.SwitchBondConfig{"bond1": {Members: []string{"ether7", "ether8"}}}},
			want: []types.PortChange{
				{Kind: types.ChangeDeleteBond, Bond: "bond1"},
				{Kind: types.ChangeCreateBond, Bond: "bond2", Ports: []string{"ether8", "ether3"}, Mode: "active-backup"},
			},
		},
		{
			name:    "new bond claiming a member of an unmanaged bond releases it",
			desired: &config.Switch{Bonds: map[string]*config.SwitchBondConfig{"bond2": {Members: []string{"ether7", "ether3"}}}},
			want: []types.PortChange{
				{Kind: types.ChangeDeleteBond, Bond: "bond1"},
				{Kind: types.ChangeCreateBond, Bond: "bond2", Ports: []string{"ether7", "ether3"}, Mode: "802.3ad"},
			},
		},
		{
			name:    "unknown port",
			desired: &config.Switch{Ports: map[string]*config.SwitchPortConfig{"ether9": access(10)}},
			err:     "port ether9 not found",
		},
		{
			name:    "uplink is changed like any port",
			desired: &config.Switch{Uplink: "sfp-sfpplus1", Ports: map[string]*config.SwitchPortConfig{"sfp-sfpplus1": trunk(0, 10)}},
			want:    []types.PortChange{{Kind: types.ChangeSetTrunk, Ports: []string{"sfp-sfpplus1"}, Tagged: []int{10}}},
		},
	}

	for _, test := range cases {
		got, err := Changes(test.desired, test.previous, liveState())
		if test.err != "" {
			if err == nil || !strings.Contains(err.Error(), test.err) {
				t.Errorf("%s: expected error %q, got %v", test.name, test.err, err)
			}

			continue
		}

		if err != nil {
			t.Errorf("%s: unexpected error %v", test.name, err)
			continue
		}

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("%s:\n got %+v\nwant %+v", test.name, got, test.want)
		}
	}
}

func TestChangesNeedVLANFiltering(t *testing.T) {
	state := liveState()
	state.VLANFiltering = false
	if _, err := Changes(&config.Switch{}, nil, state); !errors.Is(err, ErrVLANFilteringOff) {
		t.Fatalf("unexpected error %v", err)
	}
}
