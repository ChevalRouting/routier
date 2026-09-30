package ipcalc

import (
	"reflect"
	"testing"
)

func TestSubnetIpcalcExample(t *testing.T) {
	info, err := Subnet("100.64.12.192/27")
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"Address":   "100.64.12.192",
		"Netmask":   "255.255.255.224",
		"Wildcard":  "0.0.0.31",
		"Network":   "100.64.12.192/27",
		"HostMin":   "100.64.12.193",
		"HostMax":   "100.64.12.222",
		"Broadcast": "100.64.12.223",
		"Hosts":     "30",
		"Class":     "A",
	}
	got := map[string]string{
		"Address":   info.Address,
		"Netmask":   info.Netmask,
		"Wildcard":  info.Wildcard,
		"Network":   info.Network,
		"HostMin":   info.HostMin,
		"HostMax":   info.HostMax,
		"Broadcast": info.Broadcast,
		"Hosts":     info.Hosts,
		"Class":     info.Class,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}

	if FormatBinary(info.AddressBits, info.Prefix, true) != "01100100.01000000.00001100.110 00000" {
		t.Errorf("address bits = %q", FormatBinary(info.AddressBits, info.Prefix, true))
	}
	if FormatBinary(info.NetmaskBits, info.Prefix, true) != "11111111.11111111.11111111.111 00000" {
		t.Errorf("netmask bits = %q", FormatBinary(info.NetmaskBits, info.Prefix, true))
	}
	if FormatBinary(info.BroadcastBits, info.Prefix, true) != "01100100.01000000.00001100.110 11111" {
		t.Errorf("broadcast bits = %q", FormatBinary(info.BroadcastBits, info.Prefix, true))
	}
}

func TestSubnetEdgeMasks(t *testing.T) {
	p31, _ := Subnet("10.0.0.0/31")
	if p31.Hosts != "2" || p31.HostMin != "10.0.0.0" || p31.HostMax != "10.0.0.1" || p31.Broadcast != "" {
		t.Errorf("/31 = %+v", p31)
	}

	p32, _ := Subnet("10.0.0.5/32")
	if p32.Hosts != "1" || !p32.HostRoute || p32.Broadcast != "" || p32.HostMin != "10.0.0.5" {
		t.Errorf("/32 = %+v", p32)
	}

	p30, _ := Subnet("10.0.0.0/30")
	if p30.Hosts != "2" || p30.Broadcast != "10.0.0.3" || p30.HostMin != "10.0.0.1" || p30.HostMax != "10.0.0.2" {
		t.Errorf("/30 = %+v", p30)
	}
}

func TestSubnetV6(t *testing.T) {
	info, err := Subnet("2001:db8:abcd:12::1/64")
	if err != nil {
		t.Fatal(err)
	}

	if info.Family != "v6" || info.Network != "2001:db8:abcd:12::/64" {
		t.Errorf("network = %q", info.Network)
	}
	if info.HostMax != "2001:db8:abcd:12:ffff:ffff:ffff:ffff" {
		t.Errorf("host max = %q", info.HostMax)
	}
	if info.Class != "" || info.Broadcast != "" {
		t.Errorf("v6 should have no class/broadcast: %+v", info)
	}
	if info.Hosts != "18446744073709551616" {
		t.Errorf("hosts = %q", info.Hosts)
	}
}

func TestReverse(t *testing.T) {
	v4, _ := Reverse("100.64.12.192")
	if v4.Name != "192.12.64.100.in-addr.arpa" {
		t.Errorf("v4 name = %q", v4.Name)
	}

	v4z, _ := Reverse("100.64.12.192/24")
	if v4z.Zone != "12.64.100.in-addr.arpa" {
		t.Errorf("v4 zone = %q", v4z.Zone)
	}

	v6, _ := Reverse("2001:db8::1")
	want6 := "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa"
	if v6.Name != want6 {
		t.Errorf("v6 name = %q, want %q", v6.Name, want6)
	}

	v6z, _ := Reverse("2001:db8::/32")
	if v6z.Zone != "8.b.d.0.1.0.0.2.ip6.arpa" {
		t.Errorf("v6 zone = %q", v6z.Zone)
	}
}

func TestRange(t *testing.T) {
	r, err := Range("192.0.2.5", "192.0.2.20")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"192.0.2.5/32",
		"192.0.2.6/31",
		"192.0.2.8/29",
		"192.0.2.16/30",
		"192.0.2.20/32",
	}
	if !reflect.DeepEqual(r.CIDRs, want) {
		t.Errorf("cidrs = %v, want %v", r.CIDRs, want)
	}

	full, _ := Range("0.0.0.0", "255.255.255.255")
	if !reflect.DeepEqual(full.CIDRs, []string{"0.0.0.0/0"}) {
		t.Errorf("full range = %v", full.CIDRs)
	}

	v6, _ := Range("2001:db8::", "2001:db8::ff")
	if !reflect.DeepEqual(v6.CIDRs, []string{"2001:db8::/120"}) {
		t.Errorf("v6 range = %v", v6.CIDRs)
	}
}
