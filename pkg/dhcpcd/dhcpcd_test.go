//go:build linux

package dhcpcd

import (
	"fmt"
	"testing"
)

func captureSysctl(t *testing.T) map[string]string {
	t.Helper()

	got := map[string]string{}
	prev := writeSysctl
	writeSysctl = func(path, val string) error {
		got[path] = val
		return nil
	}
	t.Cleanup(func() { writeSysctl = prev })

	return got
}

func TestSyncSLAACEnables(t *testing.T) {
	got := captureSysctl(t)

	syncSLAAC("eth0", false)

	if v := got[fmt.Sprintf(sysctlAcceptRA, "eth0")]; v != slaacAcceptRA+"\n" {
		t.Fatalf("accept_ra = %q, want %q", v, slaacAcceptRA+"\n")
	}

	if v := got[fmt.Sprintf(sysctlAutoconf, "eth0")]; v != slaacAutoconf+"\n" {
		t.Fatalf("autoconf = %q, want %q", v, slaacAutoconf+"\n")
	}
}

func TestDisableSLAACSetsAcceptRAOff(t *testing.T) {
	got := captureSysctl(t)

	disableSLAAC("eth0", false)

	if v := got[fmt.Sprintf(sysctlAcceptRA, "eth0")]; v != offAcceptRA+"\n" {
		t.Fatalf("accept_ra = %q, want %q", v, offAcceptRA+"\n")
	}

	if v := got[fmt.Sprintf(sysctlAutoconf, "eth0")]; v != offAutoconf+"\n" {
		t.Fatalf("autoconf = %q, want %q", v, offAutoconf+"\n")
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	got := captureSysctl(t)

	syncSLAAC("eth0", true)
	disableSLAAC("eth1", true)

	if len(got) != 0 {
		t.Fatalf("dry run wrote %d sysctls, want 0", len(got))
	}
}
