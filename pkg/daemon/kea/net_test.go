package kea

import "testing"

func TestIPToU32(t *testing.T) {
	cases := map[string]uint32{
		"0.0.0.0":         0,
		"255.255.255.255": 0xFFFFFFFF,
		"10.0.0.1":        0x0A000001,
		"192.168.1.1":     0xC0A80101,
		"bad":             0,
	}
	for ip, want := range cases {
		if got := ipToU32(ip); got != want {
			t.Errorf("ipToU32(%q) = %#x, want %#x", ip, got, want)
		}
	}
}

func TestIPInCIDR(t *testing.T) {
	cases := []iPInCIDRCase{
		{"10.0.0.5", "10.0.0.0/24", true},
		{"10.0.1.5", "10.0.0.0/24", false},
		{"10.0.1.5", "10.0.0.0/16", true},
		{"2001:db8::1", "2001:db8::/32", true},
		{"2001:db8::1", "2001:db9::/32", false},
		{"2001:db8:0:1::5", "2001:db8::/64", false},
		{"2001:db8:0:1::5", "2001:db8::/48", true},
		{"10.0.0.5", "2001:db8::/32", false},
		{"10.0.0.5", "bad", false},
	}
	for _, c := range cases {
		if got := ipInCIDR(c.ip, c.cidr); got != c.want {
			t.Errorf("ipInCIDR(%q,%q) = %v, want %v", c.ip, c.cidr, got, c.want)
		}
	}
}

func TestIsMAC(t *testing.T) {
	cases := map[string]bool{
		"aa:bb:cc:dd:ee:ff": true,
		"AA:BB:CC:DD:EE:FF": true,
		"aa:bb:cc:dd:ee":    false,
		"aabbccddeeff":      false,
		"00:11:22:33:44:5":  false,
	}
	for id, want := range cases {
		if got := isMAC(id); got != want {
			t.Errorf("isMAC(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestParseCIDR(t *testing.T) {
	cases := []parseCIDRCase{
		{"100.64.12.12/24", "100.64.12.0/24", "100.64.12.12"},
		{"100.64.12.0/24", "100.64.12.0/24", ""},
		{"100.64.12.12", "", "100.64.12.12"},
		{"2001:db8::1", "", "2001:db8::1"},
		{"2001:db8::/32", "2001:db8::/32", ""},
		{"2001:db8::5/64", "2001:db8::/64", "2001:db8::5"},
	}
	for _, c := range cases {
		net, host := parseCIDR(c.in)
		if net != c.net || host != c.hostWant {
			t.Errorf("parseCIDR(%q) = (%q,%q), want (%q,%q)", c.in, net, host, c.net, c.hostWant)
		}
	}
}

func TestNextAvailableIP(t *testing.T) {
	got, err := nextAvailableIP("10.0.0.0/24", []string{"10.0.0.1", "10.0.0.2"})
	if err != nil {
		t.Fatal(err)
	}

	if got != "10.0.0.3" {
		t.Errorf("nextAvailableIP = %q, want 10.0.0.3", got)
	}

	if _, err := nextAvailableIP("10.0.0.0/31", []string{"10.0.0.1"}); err == nil {
		t.Error("expected exhaustion error for /31 with used address")
	}

	if _, err := nextAvailableIP("2001:db8::/64", nil); err == nil {
		t.Error("expected error for IPv6 auto-selection")
	}
}

type iPInCIDRCase struct {
	ip, cidr string
	want     bool
}

type parseCIDRCase struct {
	in            string
	net, hostWant string
}
