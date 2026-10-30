//go:build linux

package motd

import (
	"net"
	"testing"

	vnl "github.com/vishvananda/netlink"
)

func addr(t *testing.T, cidr string) vnl.Addr {
	t.Helper()
	a, err := vnl.ParseAddr(cidr)
	if err != nil {
		t.Fatal(err)
	}

	return *a
}

func TestInterfaceLine(t *testing.T) {
	vlan := &vnl.Vlan{LinkAttrs: vnl.LinkAttrs{Name: "eth0.10"}, VlanId: 10}
	if got, want := interfaceLine(vlan, []vnl.Addr{addr(t, "10.0.10.1/24"), addr(t, "fe80::1/64"), addr(t, "2001:db8::1/64")}), "eth0.10       10.0.10.1/24  2001:db8::1/64"; got != want {
		t.Fatalf("vlan line = %q, want %q", got, want)
	}

	bridge := &vnl.Bridge{LinkAttrs: vnl.LinkAttrs{Name: "br0"}}
	if got := interfaceLine(bridge, []vnl.Addr{addr(t, "192.0.2.1/24")}); got == "" {
		t.Fatal("expected bridge with an address to be listed")
	}

	eth := &vnl.Device{LinkAttrs: vnl.LinkAttrs{Name: "eth1"}}
	if got := interfaceLine(eth, []vnl.Addr{addr(t, "fe80::2/64")}); got != "" {
		t.Fatalf("link-local only interface listed: %q", got)
	}

	if got := interfaceLine(eth, nil); got != "" {
		t.Fatalf("interface without address listed: %q", got)
	}

	lo := &vnl.Device{LinkAttrs: vnl.LinkAttrs{Name: "lo", Flags: net.FlagLoopback}}
	if got := interfaceLine(lo, []vnl.Addr{addr(t, "127.0.0.1/8")}); got != "" {
		t.Fatalf("loopback listed: %q", got)
	}
}
