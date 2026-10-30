package routeros

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/types"
)

func describe(t *testing.T, d *Driver, changes ...types.PortChange) []string {
	t.Helper()

	steps, err := d.Plan(context.Background(), types.SwitchPlan{Changes: changes})
	if err != nil {
		t.Fatal(err)
	}

	lines := make([]string, 0, len(steps))
	for _, step := range steps {
		lines = append(lines, step.Describe())
	}

	return lines
}

func expectSteps(t *testing.T, got, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected steps:\n got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func TestPlanSetAccess(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d, types.PortChange{Kind: types.ChangeSetAccess, Ports: []string{"ether1"}, VLAN: 20})
	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/port/*0 frame-types="admit-only-untagged-and-priority-tagged" pvid="20"`,
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1,bond1" untagged="ether3,ether1"`,
	})
}

func TestPlanSetTrunkSplitsRanges(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d, types.PortChange{Kind: types.ChangeSetTrunk, Ports: []string{"ether3"}, Tagged: []int{10, 101}})
	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1,bond1" untagged="" vlan-ids="20"`,
		`PATCH /rest/interface/bridge/vlan/*3 tagged="sfp-sfpplus1" untagged="" vlan-ids="100,102"`,
		`PATCH /rest/interface/bridge/port/*2 frame-types="admit-only-vlan-tagged" pvid="1"`,
		`PATCH /rest/interface/bridge/vlan/*1 tagged="bridge,ether2,sfp-sfpplus1,ether3" untagged=""`,
		`PUT /rest/interface/bridge/vlan bridge="bridge" tagged="sfp-sfpplus1,ether3" untagged="" vlan-ids="101"`,
	})
}

func TestPlanTrunkWithNativeVLAN(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d, types.PortChange{Kind: types.ChangeSetTrunk, Ports: []string{"ether1"}, VLAN: 10, Tagged: []int{10, 20}})
	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/port/*0 frame-types="admit-all" pvid="10"`,
		`PATCH /rest/interface/bridge/vlan/*1 tagged="bridge,ether2,sfp-sfpplus1" untagged="ether1"`,
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1,bond1,ether1" untagged="ether3"`,
	})
}

func TestPlanCreateBond(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d, types.PortChange{Kind: types.ChangeCreateBond, Bond: "bond2", Ports: []string{"ether2", "ether3"}})
	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/vlan/*1 tagged="bridge,sfp-sfpplus1" untagged="" vlan-ids="10"`,
		`PATCH /rest/interface/bridge/vlan/*2 tagged="sfp-sfpplus1,bond1" untagged="" vlan-ids="20"`,
		`DELETE /rest/interface/bridge/port/*1`,
		`DELETE /rest/interface/bridge/port/*2`,
		`PUT /rest/interface/bonding mode="802.3ad" name="bond2" slaves="ether2,ether3"`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-all" interface="bond2" pvid="1"`,
		`PATCH /rest/interface/bridge/vlan/*1 tagged="bridge,sfp-sfpplus1,bond2" untagged=""`,
		`PATCH /rest/interface/bridge/vlan/*2 tagged="sfp-sfpplus1,bond1,bond2" untagged=""`,
	})
}

func TestPlanDeleteBond(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d, types.PortChange{Kind: types.ChangeDeleteBond, Bond: "bond1"})
	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1" untagged="ether3" vlan-ids="20"`,
		`DELETE /rest/interface/bridge/port/*3`,
		`DELETE /rest/interface/bonding/*B`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-only-vlan-tagged" interface="ether7" pvid="1"`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-only-vlan-tagged" interface="ether8" pvid="1"`,
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1,ether7,ether8" untagged="ether3"`,
	})
}

func TestPlanStepsRunAgainstTheDevice(t *testing.T) {
	d, fake := newFakeDriver(t, fakePassword)
	steps, err := d.Plan(context.Background(), types.SwitchPlan{Changes: []types.PortChange{{Kind: types.ChangeSetAccess, Ports: []string{"ether1"}, VLAN: 20}}})
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range steps {
		if err := step.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	calls := fake.recorded()
	if len(calls) != 2 || calls[0].Method != "PATCH" || calls[0].Path != "/interface/bridge/port/*0" || calls[0].Body["pvid"] != "20" {
		t.Fatalf("unexpected calls %+v", calls)
	}

	if calls[1].Path != "/interface/bridge/vlan/*2" || calls[1].Body["untagged"] != "ether3,ether1" {
		t.Fatalf("unexpected vlan call %+v", calls[1])
	}
}

func TestPlanCreateBondThenConfigureIt(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d,
		types.PortChange{Kind: types.ChangeCreateBond, Bond: "bond2", Ports: []string{"ether2", "ether3"}},
		types.PortChange{Kind: types.ChangeSetAccess, Ports: []string{"bond2"}, VLAN: 30},
	)
	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/vlan/*1 tagged="bridge,sfp-sfpplus1" untagged="" vlan-ids="10"`,
		`PATCH /rest/interface/bridge/vlan/*2 tagged="sfp-sfpplus1,bond1" untagged="" vlan-ids="20"`,
		`DELETE /rest/interface/bridge/port/*1`,
		`DELETE /rest/interface/bridge/port/*2`,
		`PUT /rest/interface/bonding mode="802.3ad" name="bond2" slaves="ether2,ether3"`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-only-untagged-and-priority-tagged" interface="bond2" pvid="30"`,
		`PUT /rest/interface/bridge/vlan bridge="bridge" tagged="" untagged="bond2" vlan-ids="30"`,
	})
}

func TestPlanDeleteBondThenConfigureMembers(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d,
		types.PortChange{Kind: types.ChangeDeleteBond, Bond: "bond1"},
		types.PortChange{Kind: types.ChangeSetAccess, Ports: []string{"ether7"}, VLAN: 10},
	)
	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1" untagged="ether3" vlan-ids="20"`,
		`DELETE /rest/interface/bridge/port/*3`,
		`DELETE /rest/interface/bonding/*B`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-only-untagged-and-priority-tagged" interface="ether7" pvid="10"`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-only-vlan-tagged" interface="ether8" pvid="1"`,
		`PATCH /rest/interface/bridge/vlan/*1 tagged="bridge,ether2,sfp-sfpplus1" untagged="ether7"`,
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1,ether8" untagged="ether3"`,
	})
}

func TestPlanDeleteBondThenReuseAMember(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d,
		types.PortChange{Kind: types.ChangeDeleteBond, Bond: "bond1"},
		types.PortChange{Kind: types.ChangeCreateBond, Bond: "bond2", Ports: []string{"ether8", "ether3"}},
	)
	expectSteps(t, got, []string{
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1" untagged="" vlan-ids="20"`,
		`DELETE /rest/interface/bridge/port/*3`,
		`DELETE /rest/interface/bridge/port/*2`,
		`DELETE /rest/interface/bonding/*B`,
		`PUT /rest/interface/bonding mode="802.3ad" name="bond2" slaves="ether8,ether3"`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-only-vlan-tagged" interface="ether7" pvid="1"`,
		`PUT /rest/interface/bridge/port bridge="bridge" frame-types="admit-all" interface="bond2" pvid="1"`,
		`PATCH /rest/interface/bridge/vlan/*2 tagged="ether2,sfp-sfpplus1,ether7,bond2" untagged=""`,
	})
}

func TestPlanDeletesAVLANEntryLeftEmpty(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	got := describe(t, d, types.PortChange{Kind: types.ChangeSetAccess, Ports: []string{"ether3"}, VLAN: 10},
		types.PortChange{Kind: types.ChangeSetTrunk, Ports: []string{"ether2", "sfp-sfpplus1"}, Tagged: []int{10}},
		types.PortChange{Kind: types.ChangeSetTrunk, Ports: []string{"bond1"}, Tagged: []int{10}},
	)
	if !slices.Contains(got, `DELETE /rest/interface/bridge/vlan/*2`) {
		t.Fatalf("empty vlan 20 entry not deleted:\n  %s", strings.Join(got, "\n  "))
	}
}
