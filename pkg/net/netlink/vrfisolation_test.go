//go:build linux

package netlink

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestKeaBindsInVRFs(t *testing.T) {
	keys := []string{"net.ipv4.ip_nonlocal_bind", "net.ipv6.ip_nonlocal_bind"}
	cfg := &config.Config{
		Interfaces: map[string]*config.Interface{"lab": {Select: "eth1", VRF: "blue"}, "lan": {Select: "eth0"}},
		DHCP:       &config.DHCP{Enabled: true, Interfaces: []string{"lan"}},
	}

	root := fakeSysctls(t, keys, "0")
	bindServicesInVRFs(cfg, root, false)
	if sysctlValue(t, root, keys[0]) != "0" {
		t.Fatal("dhcp outside vrfs needs nothing")
	}

	cfg.DHCP.Interfaces = append(cfg.DHCP.Interfaces, "lab")
	bindServicesInVRFs(cfg, root, false)
	if sysctlValue(t, root, keys[0]) != "1" || sysctlValue(t, root, keys[1]) != "1" {
		t.Fatal("kea must bind the address of an interface in a vrf")
	}
}

func TestVRFTablesAreClosed(t *testing.T) {
	enterTestNetns(t)
	cfg := &config.Config{VRFs: map[string]*config.VRFConfig{"blue": {Table: 100}}}
	for i := 0; i < 2; i++ {
		if err := ReconcileLinks(cfg, false); err != nil {
			t.Fatal(err)
		}
	}

	for _, family := range []int{vnl.FAMILY_V4, vnl.FAMILY_V6} {
		routes, err := vnl.RouteListFiltered(family, &vnl.Route{Table: 100}, vnl.RT_FILTER_TABLE)
		if err != nil {
			t.Fatal(err)
		}

		closed := false
		for _, route := range routes {
			if route.Type == unix.RTN_UNREACHABLE && route.Priority == vrfUnreachableMetric {
				closed = true
			}
		}

		if !closed {
			t.Fatalf("family %d: no unreachable default in the vrf table: %+v", family, routes)
		}

		rules, err := vnl.RuleList(family)
		if err != nil {
			t.Fatal(err)
		}

		for _, rule := range rules {
			if rule.Table == localTable && rule.Priority == 0 {
				t.Fatalf("family %d: the local rule still comes before the vrf rule", family)
			}
		}
	}

	delete(cfg.VRFs, "blue")
	if err := ReconcileLinks(cfg, false); err != nil {
		t.Fatal(err)
	}

	routes, err := vnl.RouteListFiltered(vnl.FAMILY_V4, &vnl.Route{Table: 100}, vnl.RT_FILTER_TABLE)
	if err != nil || len(routes) != 0 {
		t.Fatalf("a removed vrf leaves its table: %+v %v", routes, err)
	}
}
