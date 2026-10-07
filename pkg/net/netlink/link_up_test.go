//go:build linux

package netlink

import (
	"errors"
	vnl "github.com/vishvananda/netlink"
	"testing"
)

func TestBringUpLinksParentsFirst(t *testing.T) {
	desired := map[string]linkSpec{"eth0": {device: "eth0"}, "isp": {device: "isp", parent: "eth0"}, "inner": {device: "inner", parent: "isp"}}
	actual := map[string]vnl.Link{}
	for name := range desired {
		actual[name] = &vnl.Dummy{LinkAttrs: vnl.LinkAttrs{Name: name}}
	}

	for i := 0; i < 100; i++ {
		var order []string
		err := bringUpLinks(desired, actual, false, func(l vnl.Link) error { order = append(order, l.Attrs().Name); return nil })
		if err != nil {
			t.Fatal(err)
		}

		if len(order) != 3 || order[0] != "eth0" || order[1] != "isp" || order[2] != "inner" {
			t.Fatalf("wrong order: %v", order)
		}
	}

	want := errors.New("network is down")
	calls := 0
	if err := bringUpLinks(desired, actual, false, func(vnl.Link) error { calls++; return want }); !errors.Is(err, want) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}

	if err := bringUpLinks(desired, actual, true, func(vnl.Link) error { t.Fatal("dry run mutated link"); return nil }); err != nil {
		t.Fatal(err)
	}
}
