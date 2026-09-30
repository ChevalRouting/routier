//go:build linux && integration

package systemtest

import (
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/netlink"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestBondAddsMissingMemberAndReapplies(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	original, err := os.Open("/proc/self/task/" + strconv.Itoa(unix.Gettid()) + "/ns/net")
	if err != nil {
		t.Fatal(err)
	}

	defer closeNetlinkNamespace(original)
	if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
		t.Skipf("network namespace requires CAP_SYS_ADMIN: %v", err)
	}

	defer restoreNetlinkNamespace(t, original)
	for _, name := range []string{"port1", "port2"} {
		if err := vnl.LinkAdd(&vnl.Dummy{LinkAttrs: vnl.LinkAttrs{Name: name}}); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{Interfaces: map[string]*config.Interface{
		"core":  {Type: "bond", Device: "core", Bond: &config.Bond{Mode: "802.3ad", MIIMon: 100, MemberDevices: []string{"port1"}}},
		"core1": {Device: "port1"},
		"core2": {Device: "port2"},
	}}
	if err := netlink.ReconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	cfg.Interfaces["core"].Bond.MemberDevices = []string{"port1", "port2"}
	for i := 0; i < 3; i++ {
		if err := netlink.ReconcileLinks(cfg, false); err != nil {
			t.Fatal(err)
		}

		bond, err := vnl.LinkByName("core")
		if err != nil {
			t.Fatal(err)
		}

		for _, name := range []string{"port1", "port2"} {
			member, err := vnl.LinkByName(name)
			if err != nil {
				t.Fatal(err)
			}

			if member.Attrs().MasterIndex != bond.Attrs().Index {
				t.Fatalf("%s missing from core on apply %d", name, i)
			}
		}
	}

	cfg.Interfaces["core"].Bond.MemberDevices = []string{"port1", "missing"}
	if err := netlink.ReconcileLinks(cfg, false); err == nil {
		t.Fatal("missing member reported success")
	}

	cfg.Interfaces["core"].Bond.MemberDevices = []string{"core"}
	if err := netlink.ReconcileLinks(cfg, false); err == nil {
		t.Fatal("self-enslavement reported success")
	}
}

func closeNetlinkNamespace(namespace *os.File) {
	_ = namespace.Close()
}

func restoreNetlinkNamespace(t *testing.T, namespace *os.File) {
	t.Helper()
	if err := unix.Setns(int(namespace.Fd()), unix.CLONE_NEWNET); err != nil {
		t.Fatalf("restore namespace: %v", err)
	}
}
