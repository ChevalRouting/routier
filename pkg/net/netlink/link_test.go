//go:build linux

package netlink

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestMarkManagedVXLAN(t *testing.T) {
	link := &vnl.Vxlan{LinkAttrs: vnl.LinkAttrs{Name: "vxlan-old"}}
	wantErr := errors.New("alias failed")
	calls := 0
	setAlias := func(unusedArg3 vnl.Link, alias string) error {
		return testMarkManagedVXLANCallback(t, wantErr, &calls, unusedArg3, alias)
	}
	if err := markManagedVXLAN(link, true, setAlias); err != nil || calls != 0 {
		t.Fatalf("dry run: err=%v calls=%d", err, calls)
	}

	if err := markManagedVXLAN(link, false, setAlias); !errors.Is(err, wantErr) {
		t.Fatalf("want alias failure, got %v", err)
	}

	link.Alias = managedAlias
	if err := markManagedVXLAN(link, false, setAlias); err != nil || calls != 1 {
		t.Fatalf("already managed: err=%v calls=%d", err, calls)
	}
}

func TestVXLANRenameRemovesPreviousLink(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	original, err := os.Open("/proc/self/task/" + strconv.Itoa(unix.Gettid()) + "/ns/net")
	if err != nil {
		t.Fatal(err)
	}

	defer func(action func() error) { _ = action() }(original.Close)
	if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
		t.Skipf("network namespace requires CAP_SYS_ADMIN: %v", err)
	}

	defer func() {
		if err := unix.Setns(int(original.Fd()), unix.CLONE_NEWNET); err != nil {
			t.Fatalf("restore network namespace: %v", err)
		}
	}()
	cfg := &config.Config{Interfaces: map[string]*config.Interface{
		"old": {Type: "vxlan", Device: "vxlan-old", VXLAN: &config.VXLAN{VNI: 100}},
	}}
	if err := reconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	old, err := vnl.LinkByName("vxlan-old")
	if err != nil || old.Attrs().Alias != managedAlias {
		t.Fatalf("old VXLAN must be managed: link=%v err=%v", old, err)
	}

	cfg.Interfaces["old"].Device = "vxlan-new"
	if err := reconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	if _, err := vnl.LinkByName("vxlan-old"); err == nil {
		t.Fatal("previous VXLAN still exists")
	}

	if _, err := vnl.LinkByName("vxlan-new"); err != nil {
		t.Fatalf("renamed VXLAN missing: %v", err)
	}
}

func testMarkManagedVXLANCallback(t *testing.T, wantErr error, calls *int, _ vnl.Link, alias string) error {
	(*calls)++
	if alias != managedAlias {
		t.Fatalf("alias = %q", alias)
	}

	return wantErr
}
