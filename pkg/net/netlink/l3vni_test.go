//go:build linux

package netlink

import (
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func enterTestNetns(t *testing.T) {
	t.Helper()
	runtime.LockOSThread()
	original, err := os.Open("/proc/self/task/" + strconv.Itoa(unix.Gettid()) + "/ns/net")
	if err != nil {
		t.Fatal(err)
	}

	if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
		_ = original.Close()
		runtime.UnlockOSThread()
		t.Skipf("network namespace requires CAP_SYS_ADMIN: %v", err)
	}

	t.Cleanup(func() { leaveTestNetns(t, original) })
}

func leaveTestNetns(t *testing.T, original *os.File) {
	defer runtime.UnlockOSThread()
	defer func() { _ = original.Close() }()
	if err := unix.Setns(int(original.Fd()), unix.CLONE_NEWNET); err != nil {
		t.Fatal(err)
	}
}

func TestL3VNILifecycle(t *testing.T) {
	enterTestNetns(t)
	cfg := &config.Config{
		Interfaces: map[string]*config.Interface{
			"fabric":  {Type: "vxlan-bridge", Device: "fabric", MTU: 1450, VXLAN: &config.VXLAN{Local: "192.0.2.10"}},
			"servers": {Type: "vnet", Device: "servers", Select: "fabric", VRF: "blue", VNet: &config.VNet{VNI: 10200, VLAN: 200}},
		},
		VRFs: map[string]*config.VRFConfig{"blue": {Table: 100, L3VNI: &config.VRFL3VNI{VNI: 5000, Transport: "fabric", VLAN: 3999}}},
	}

	for i := 0; i < 2; i++ {
		if err := ReconcileLinks(cfg, false); err != nil {
			t.Fatal(err)
		}
	}

	svi, err := vnl.LinkByName(config.L3VNILinkName("blue"))
	if err != nil {
		t.Fatal(err)
	}

	fabric, err := vnl.LinkByName("fabric")
	if err != nil {
		t.Fatal(err)
	}

	blue, err := vnl.LinkByName("blue")
	if err != nil {
		t.Fatal(err)
	}

	vlan, ok := svi.(*vnl.Vlan)
	if !ok || vlan.VlanId != 3999 || vlan.ParentIndex != fabric.Attrs().Index || svi.Attrs().MasterIndex != blue.Attrs().Index {
		t.Fatalf("l3vni svi %+v", svi.Attrs())
	}

	vx, err := vnl.LinkByName(config.VNetLinkName("fabric", "x"))
	if err != nil {
		t.Fatal(err)
	}

	assertTunnels(t, vx, map[uint16]uint32{200: 10200, 3999: 5000})
	if !bridgeCarries(t, fabric, 3999) {
		t.Fatal("the transport bridge must carry the l3vni vlan itself")
	}

	cfg.VRFs["blue"].L3VNI = nil
	if err := ReconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	if _, err := vnl.LinkByName(config.L3VNILinkName("blue")); err == nil {
		t.Fatal("stale l3vni svi remains")
	}

	assertTunnels(t, vx, map[uint16]uint32{200: 10200})
}

func assertTunnels(t *testing.T, vx vnl.Link, want map[uint16]uint32) {
	t.Helper()
	tunnels, err := vnl.BridgeVlanTunnelShowDev(vx)
	if err != nil || len(tunnels) != len(want) {
		t.Fatalf("tunnel mappings: %v, %v", tunnels, err)
	}

	for _, tunnel := range tunnels {
		if want[tunnel.Vid] != tunnel.TunId {
			t.Fatalf("tunnel mappings: %v", tunnels)
		}
	}
}

func bridgeCarries(t *testing.T, bridge vnl.Link, vid uint16) bool {
	t.Helper()
	vlans, err := vnl.BridgeVlanList()
	if err != nil {
		t.Fatal(err)
	}

	for _, vlan := range vlans[int32(bridge.Attrs().Index)] {
		if vlan.Vid == vid {
			return true
		}
	}

	return false
}
