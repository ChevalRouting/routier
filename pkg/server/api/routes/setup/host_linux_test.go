//go:build linux

package setup

import "testing"

func TestPidfileIface(t *testing.T) {
	cases := map[string]string{
		"eth0-4.pid":      "eth0",
		"wan.100-4.pid":   "wan.100",
		"udhcpc.eth1.pid": "eth1",
		"eth2.pid":        "eth2",
		"eth0-6.pid":      "",
	}
	for base, want := range cases {
		if got := pidfileIface(base); got != want {
			t.Errorf("pidfileIface(%q) = %q, want %q", base, got, want)
		}
	}
}
