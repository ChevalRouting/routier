package routeros

import (
	"context"
	"net"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/switches/driver"

	"github.com/ChevalRouting/routier/pkg/types"
)

func portByName(state types.SwitchState, name string) types.SwitchPort {
	for _, port := range state.Ports {
		if port.Name == name {
			return port
		}
	}

	return types.SwitchPort{}
}

func TestStateParsesRouterOS(t *testing.T) {
	d, _ := newFakeDriver(t, fakePassword)
	state, err := d.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if state.Facts.Identity != "crs310-lab" || state.Facts.Model != "CRS310-8G+2S+" || state.Facts.Serial != "HJ0123456" || state.Facts.Version != "7.16 (stable)" {
		t.Fatalf("unexpected facts %+v", state.Facts)
	}

	if !state.VLANFiltering || !slices.Equal(state.Bridges, []string{"bridge"}) {
		t.Fatalf("unexpected bridges %+v", state)
	}

	if len(state.Ports) != 8 {
		t.Fatalf("expected ether and bond ports only, got %d", len(state.Ports))
	}

	access := portByName(state, "ether1")
	if access.Mode != types.PortAccess || access.PVID != 10 || !slices.Equal(access.Untagged, []int{10}) || access.Rate != "1Gbps" || access.Comment != "desk" {
		t.Fatalf("unexpected access port %+v", access)
	}

	trunk := portByName(state, "ether2")
	if trunk.Mode != types.PortTrunk || trunk.PVID != 1 || !slices.Equal(trunk.Tagged, []int{10, 20}) || !slices.Equal(trunk.Untagged, []int{1}) {
		t.Fatalf("unexpected trunk port %+v", trunk)
	}

	uplink := portByName(state, "sfp-sfpplus1")
	if uplink.Mode != types.PortTrunk || !slices.Equal(uplink.Tagged, []int{10, 20, 100, 101, 102}) {
		t.Fatalf("unexpected uplink %+v", uplink)
	}

	member := portByName(state, "ether7")
	if member.Mode != types.PortBondSlave || member.Bond != "bond1" {
		t.Fatalf("unexpected bond member %+v", member)
	}

	bond := portByName(state, "bond1")
	if bond.Mode != types.PortTrunk || !slices.Equal(bond.Tagged, []int{20}) {
		t.Fatalf("unexpected bond port %+v", bond)
	}

	if disabled := portByName(state, "sfp-sfpplus2"); disabled.Mode != types.PortUnbridged || !disabled.Disabled {
		t.Fatalf("unexpected unbridged port %+v", disabled)
	}

	if len(state.Bonds) != 1 || !slices.Equal(state.Bonds[0].Members, []string{"ether7", "ether8"}) || state.Bonds[0].Mode != "802.3ad" {
		t.Fatalf("unexpected bonds %+v", state.Bonds)
	}

	ids := make([]int, 0, len(state.VLANs))
	for _, vlan := range state.VLANs {
		ids = append(ids, vlan.ID)
	}

	if !slices.Equal(ids, []int{1, 10, 20, 100, 101, 102}) {
		t.Fatalf("unexpected vlans %v", ids)
	}

	if !state.VLANs[0].Dynamic || state.VLANs[1].Dynamic {
		t.Fatalf("dynamic flag not kept: %+v", state.VLANs[:2])
	}
}

func TestStateReportsAuthErrors(t *testing.T) {
	d, _ := newFakeDriver(t, "wrong")
	_, err := d.State(context.Background())
	if err == nil || !strings.Contains(err.Error(), "HTTP 401: Unauthorized") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestExpandIDs(t *testing.T) {
	if got := expandIDs("10, 20,100-102,bad,7-3"); !slices.Equal(got, []int{10, 20, 100, 101, 102}) {
		t.Fatalf("got %v", got)
	}
}

func TestStateOverPlainHTTP(t *testing.T) {
	server := httptest.NewServer(&fakeRouterOS{})
	t.Cleanup(server.Close)

	host, port, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}

	created, err := New(driver.Target{Host: host, Scheme: "http", Port: portNumber, Credentials: map[string]string{"username": fakeUser, "password": fakePassword}})
	if err != nil {
		t.Fatal(err)
	}

	state, err := created.State(context.Background())
	if err != nil || state.Facts.Identity != "crs310-lab" {
		t.Fatalf("plain http read failed: %+v %v", state.Facts, err)
	}
}

func TestDefaultPortFollowsTheScheme(t *testing.T) {
	cases := map[string]string{"": "https://10.0.0.2:443/rest", "https": "https://10.0.0.2:443/rest", "http": "http://10.0.0.2:80/rest"}
	for scheme, want := range cases {
		created, err := New(driver.Target{Host: "10.0.0.2", Scheme: scheme, Credentials: map[string]string{"username": "u"}})
		if err != nil {
			t.Fatal(err)
		}

		if got := created.(*Driver).client.base; got != want {
			t.Errorf("scheme %q: base %q, want %q", scheme, got, want)
		}
	}
}
