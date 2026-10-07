package kea

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"testing"
)

func TestUnixSocketExchange(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "kea4.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}

	defer func(action func() error) { _ = action() }(ln.Close)

	go func() { testUnixSocketExchangeCallback(ln) }()

	c := &Client{unix: true, endpoints: map[string]string{"dhcp4": sock}}
	subnets, err := c.Subnets("dhcp4")
	if err != nil {
		t.Fatal(err)
	}

	if len(subnets) != 1 || subnets[0].Subnet != "10.0.0.0/24" {
		t.Fatalf("unexpected subnets over unix socket: %+v", subnets)
	}
}

type caCall struct {
	Command   string          `json:"command"`
	Service   []string        `json:"service"`
	Arguments json.RawMessage `json:"arguments"`
}

func fakeCA(t *testing.T, replies map[string]string, record *[]caCall) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fakeCACallback(t, replies, record, w, r) }))
}

func TestSubnetsAndLeases(t *testing.T) {
	srv := fakeCA(t, map[string]string{
		"config-get":     `[{"result":0,"arguments":{"Dhcp4":{"subnet4":[{"id":1,"subnet":"10.0.0.0/24"}]}}}]`,
		"lease4-get-all": `[{"result":0,"arguments":{"leases":[{"ip-address":"10.0.0.5","hw-address":"aa:bb:cc:dd:ee:ff","hostname":"host1"}]}}]`,
	}, nil)
	defer srv.Close()

	c := New(srv.URL, "", "")

	subnets, err := c.Subnets("dhcp4")
	if err != nil {
		t.Fatal(err)
	}

	if len(subnets) != 1 || subnets[0].ID != 1 || subnets[0].Subnet != "10.0.0.0/24" {
		t.Fatalf("unexpected subnets: %+v", subnets)
	}

	leases, err := c.Leases("dhcp4", 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(leases) != 1 || leases[0].IPAddress != "10.0.0.5" || leases[0].Hostname != "host1" {
		t.Fatalf("unexpected leases: %+v", leases)
	}
}

func TestSendResultError(t *testing.T) {
	srv := fakeCA(t, map[string]string{
		"config-get": `[{"result":3,"text":"empty"}]`,
	}, nil)
	defer srv.Close()

	if _, err := New(srv.URL, "", "").Subnets("dhcp4"); err == nil {
		t.Fatal("expected error from non-zero result")
	}
}

func TestAddReservationAutoIP(t *testing.T) {
	var calls []caCall
	srv := fakeCA(t, map[string]string{
		"config-get":          `[{"result":0,"arguments":{"Dhcp4":{"subnet4":[{"id":7,"subnet":"10.0.0.0/24"}]}}}]`,
		"lease4-get-all":      `[{"result":0,"arguments":{"leases":[{"ip-address":"10.0.0.1"}]}}]`,
		"reservation-get-all": `[{"result":0,"arguments":{"hosts":[{"ip-address":"10.0.0.2"}]}}]`,
		"reservation-add":     `[{"result":0,"arguments":{}}]`,
	}, &calls)
	defer srv.Close()

	c := New(srv.URL, "", "")
	ip, err := c.AddReservation("printer", "aa:bb:cc:dd:ee:ff", "10.0.0.0/24")
	if err != nil {
		t.Fatal(err)
	}

	if ip != "10.0.0.3" {
		t.Fatalf("auto-picked ip = %q, want 10.0.0.3 (0.1 leased, 0.2 reserved)", ip)
	}

	var addArgs reqReservationV4
	for _, call := range calls {
		if call.Command == "reservation-add" {
			if err := json.Unmarshal(call.Arguments, &addArgs); err != nil {
				t.Fatal(err)
			}
		}
	}

	if addArgs.Reservation.HWAddress != "aa:bb:cc:dd:ee:ff" || addArgs.Reservation.IPAddress != "10.0.0.3" {
		t.Fatalf("unexpected reservation args: %+v", addArgs.Reservation)
	}
}

func TestDelReservationIPv6(t *testing.T) {
	var calls []caCall
	srv := fakeCA(t, map[string]string{
		"config-get":      `[{"result":0,"arguments":{"Dhcp6":{"subnet6":[{"id":9,"subnet":"2001:db8::/64"}]}}}]`,
		"reservation-del": `[{"result":0,"arguments":{}}]`,
	}, &calls)
	defer srv.Close()

	c := New(srv.URL, "", "")
	if err := c.DelReservation("2001:db8::5"); err != nil {
		t.Fatalf("del v6 reservation: %v", err)
	}

	var delArgs reqSubnetIP
	found := false
	for _, call := range calls {
		if call.Command == "reservation-del" {
			found = true
			if err := json.Unmarshal(call.Arguments, &delArgs); err != nil {
				t.Fatal(err)
			}
		}
	}

	if !found {
		t.Fatal("reservation-del was never called (v6 subnet not resolved)")
	}

	if delArgs.SubnetID != 9 || delArgs.IPAddress != "2001:db8::5" {
		t.Fatalf("unexpected del args: %+v", delArgs)
	}
}

func TestAddReservationIPv6(t *testing.T) {
	var calls []caCall
	srv := fakeCA(t, map[string]string{
		"config-get":          `[{"result":0,"arguments":{"Dhcp6":{"subnet6":[{"id":9,"subnet":"2001:db8::/64"}]}}}]`,
		"lease6-get-all":      `[{"result":0,"arguments":{"leases":[]}}]`,
		"reservation-get-all": `[{"result":0,"arguments":{"hosts":[]}}]`,
		"reservation-add":     `[{"result":0,"arguments":{}}]`,
	}, &calls)
	defer srv.Close()

	c := New(srv.URL, "", "")
	ip, err := c.AddReservation("laptop", "00:03:00:01:aa:bb:cc:dd:ee:ff", "2001:db8::abc")
	if err != nil {
		t.Fatalf("add v6 reservation: %v", err)
	}

	if ip != "2001:db8::abc" {
		t.Fatalf("v6 reserved ip = %q, want 2001:db8::abc", ip)
	}

	var addArgs reqReservationV6
	for _, call := range calls {
		if call.Command == "reservation-add" {
			if err := json.Unmarshal(call.Arguments, &addArgs); err != nil {
				t.Fatal(err)
			}
		}
	}

	if addArgs.Reservation.SubnetID != 9 || addArgs.Reservation.DUID != "00:03:00:01:aa:bb:cc:dd:ee:ff" ||
		len(addArgs.Reservation.IPAddresses) != 1 || addArgs.Reservation.IPAddresses[0] != "2001:db8::abc" {
		t.Fatalf("unexpected v6 reservation args: %+v", addArgs.Reservation)
	}
}

func TestFreeIPOutOfPool(t *testing.T) {
	srv := fakeCA(t, map[string]string{
		"config-get": `[{"result":0,"arguments":{"Dhcp4":{"subnet4":[{"id":1,"subnet":"10.0.0.0/24","pools":[{"pool":"10.0.0.100-10.0.0.200"}],"reservations":[{"ip-address":"10.0.0.1"}]}]}}}]`,
	}, nil)
	defer srv.Close()

	ip, err := New(srv.URL, "", "").FreeIP("10.0.0.0/24", nil)
	if err != nil {
		t.Fatal(err)
	}

	if ip != "10.0.0.2" {
		t.Fatalf("free ip = %q, want 10.0.0.2 (outside pool, not reserved)", ip)
	}
}

func TestFreeIPSkipsExclusions(t *testing.T) {
	srv := fakeCA(t, map[string]string{
		"config-get": `[{"result":0,"arguments":{"Dhcp4":{"subnet4":[{"id":1,"subnet":"10.0.0.0/24","pools":[{"pool":"10.0.0.100-10.0.0.200"}],"reservations":[{"ip-address":"10.0.0.1"}]}]}}}]`,
	}, nil)
	defer srv.Close()

	ip, err := New(srv.URL, "", "").FreeIP("10.0.0.0/24", []string{"10.0.0.2-10.0.0.20", "10.0.0.21"})
	if err != nil {
		t.Fatal(err)
	}

	if ip != "10.0.0.22" {
		t.Fatalf("free ip = %q, want 10.0.0.22 (skips exclusions)", ip)
	}
}

func TestFreeIPv6OutOfPool(t *testing.T) {
	srv := fakeCA(t, map[string]string{
		"config-get": `[{"result":0,"arguments":{"Dhcp6":{"subnet6":[{"id":1,"subnet":"2001:db8::/64","pools":[{"pool":"2001:db8::1-2001:db8::ffff"}],"reservations":[{"ip-addresses":["2001:db8::1:5"]}]}]}}}]`,
	}, nil)
	defer srv.Close()

	c := New(srv.URL, "", "")
	subnet := netip.MustParsePrefix("2001:db8::/64")
	poolLo := netip.MustParseAddr("2001:db8::1")
	poolHi := netip.MustParseAddr("2001:db8::ffff")

	for i := 0; i < 20; i++ {
		s, err := c.FreeIP("2001:db8::/64", nil)
		if err != nil {
			t.Fatal(err)
		}

		a := netip.MustParseAddr(s)
		if !subnet.Contains(a) {
			t.Fatalf("free v6 ip %q not in subnet", s)
		}

		if a.Compare(poolLo) >= 0 && a.Compare(poolHi) <= 0 {
			t.Fatalf("free v6 ip %q is inside the pool", s)
		}

		if s == "2001:db8::1:5" {
			t.Fatalf("free v6 ip %q is the reserved address", s)
		}

		if a == subnet.Addr() {
			t.Fatalf("free v6 ip %q is the subnet-router anycast", s)
		}
	}
}

func TestBasicAuthHeader(t *testing.T) {
	got := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `[{"result":0,"arguments":{"subnets":[]}}]`)
	}))
	defer srv.Close()

	c := New(srv.URL, "kea", "secret")
	if _, err := c.Subnets("dhcp4"); err != nil {
		t.Fatal(err)
	}

	if want := basicAuth("kea", "secret"); got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
}

func TestLeaseQueryEmptyResults(t *testing.T) {
	for _, command := range []string{"lease4-get-all", "lease6-get-all", "lease4-get", "lease6-get"} {
		t.Run(command, func(t *testing.T) { testLeaseQueryEmptyResultsCallback(command, t) })
	}

	if _, err := parseResponse("lease4-get-all", []byte(`{"result":1,"text":"lease commands hook unavailable"}`)); err == nil {
		t.Fatal("lease query failures must remain errors")
	}

	if _, err := parseResponse("lease4-del", []byte(`{"result":3,"text":"No lease deleted"}`)); err == nil {
		t.Fatal("empty mutation results must remain errors")
	}
}

func testUnixSocketExchangeCallback(ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}

	defer func(action func() error) { _ = action() }(conn.Close)
	_, _ = io.ReadAll(conn)
	_, _ = io.WriteString(conn, `{"result":0,"arguments":{"Dhcp4":{"subnet4":[{"id":1,"subnet":"10.0.0.0/24"}]}}}`)
}

func fakeCACallback(t *testing.T, replies map[string]string, record *[]caCall, w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var call caCall
	if err := json.Unmarshal(body, &call); err != nil {
		t.Errorf("bad request body: %v", err)
	}

	if record != nil {
		*record = append(*record, call)
	}

	reply, ok := replies[call.Command]
	if !ok {
		reply = `[{"result":1,"text":"unknown command"}]`
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, reply)
}

func testLeaseQueryEmptyResultsCallback(command string, t *testing.T) {
	raw, err := parseResponse(command, []byte(`{"result":3,"text":"No matching leases."}`))
	if err != nil {
		t.Fatal(err)
	}

	if command == "lease4-get-all" || command == "lease6-get-all" {
		var result leaseList
		if err := json.Unmarshal(raw, &result); err != nil || len(result.Leases) != 0 {
			t.Fatalf("expected empty lease list: %s, %v", raw, err)
		}
	} else {
		var lease Lease
		if err := json.Unmarshal(raw, &lease); err != nil || lease.IPAddress != "" {
			t.Fatalf("expected empty lease: %s, %v", raw, err)
		}
	}
}
