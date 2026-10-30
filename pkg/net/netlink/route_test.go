//go:build linux

package netlink

import (
	"net"
	"testing"

	vnl "github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func TestParseRouteScope(t *testing.T) {
	cases := map[string]vnl.Scope{
		"global":   vnl.SCOPE_UNIVERSE,
		"universe": vnl.SCOPE_UNIVERSE,
		"site":     vnl.SCOPE_SITE,
		"link":     vnl.SCOPE_LINK,
		"host":     vnl.SCOPE_HOST,
	}

	for in, want := range cases {
		got, err := parseRouteScope(in)
		if err != nil {
			t.Fatalf("parseRouteScope(%q) returned error: %v", in, err)
		}

		if got != want {
			t.Errorf("parseRouteScope(%q) = %d, want %d", in, got, want)
		}
	}

	if _, err := parseRouteScope("bogus"); err == nil {
		t.Error("parseRouteScope(\"bogus\") expected error, got nil")
	}
}

func TestRoutesEqualScope(t *testing.T) {
	cur := vnl.Route{Scope: vnl.SCOPE_UNIVERSE}
	desired := vnl.Route{Scope: vnl.SCOPE_LINK}

	if routesEqual(cur, desired) {
		t.Error("routes with differing scope should not be equal")
	}

	if !routesEqual(cur, vnl.Route{}) {
		t.Error("desired with unset scope should ignore scope in comparison")
	}
}

func TestRoutesEqualSource(t *testing.T) {
	src := net.ParseIP("2001:db8::10")
	if routesEqual(vnl.Route{}, vnl.Route{Src: src}) {
		t.Error("adding a source should replace the route")
	}

	if routesEqual(vnl.Route{Src: src}, vnl.Route{}) {
		t.Error("removing a source should replace the route")
	}

	if !routesEqual(vnl.Route{Src: src}, vnl.Route{Src: net.ParseIP("2001:db8::10")}) {
		t.Error("an unchanged source should keep the route")
	}
}

func TestParseRouteType(t *testing.T) {
	cases := map[string]int{
		"":            0,
		"blackhole":   unix.RTN_BLACKHOLE,
		"unreachable": unix.RTN_UNREACHABLE,
		"prohibit":    unix.RTN_PROHIBIT,
	}

	for in, want := range cases {
		got, err := parseRouteType(in)
		if err != nil {
			t.Fatalf("parseRouteType(%q) returned error: %v", in, err)
		}

		if got != want {
			t.Errorf("parseRouteType(%q) = %d, want %d", in, got, want)
		}
	}

	if _, err := parseRouteType("bogus"); err == nil {
		t.Error("parseRouteType(\"bogus\") expected error, got nil")
	}
}

func TestRouteKeyType(t *testing.T) {
	_, dst, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}

	blackhole := routeKey(vnl.Route{Dst: dst, Type: unix.RTN_BLACKHOLE})
	if blackhole != "10.0.0.0/8 blackhole" {
		t.Errorf("blackhole key = %q", blackhole)
	}

	if unicast := routeKey(vnl.Route{Dst: dst, Type: unix.RTN_UNICAST}); unicast != "10.0.0.0/8" {
		t.Errorf("unicast key = %q", unicast)
	}
}
