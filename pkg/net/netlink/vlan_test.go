//go:build linux

package netlink

import (
	"testing"

	vnl "github.com/vishvananda/netlink"
)

func TestVLANMovedOnlyForManagedVLANs(t *testing.T) {
	parent := &vnl.Device{LinkAttrs: vnl.LinkAttrs{Index: 3}}
	managed := &vnl.Vlan{LinkAttrs: vnl.LinkAttrs{ParentIndex: 2, Alias: managedAlias}, VlanId: 40}
	if !vlanMoved(managed, parent, 40) {
		t.Fatal("a managed vlan on another parent must be recreated")
	}

	managed.ParentIndex = 3
	if vlanMoved(managed, parent, 40) {
		t.Fatal("same parent and id must be kept")
	}

	if !vlanMoved(managed, parent, 41) {
		t.Fatal("another vlan id must be recreated")
	}

	foreign := &vnl.Vlan{LinkAttrs: vnl.LinkAttrs{ParentIndex: 2}, VlanId: 40}
	if vlanMoved(foreign, parent, 40) {
		t.Fatal("a vlan routier did not create is left alone")
	}
}
