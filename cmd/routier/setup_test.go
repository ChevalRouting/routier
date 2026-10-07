package main

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestAskInterfacesSupportsNone(t *testing.T) {
	tty := t.TempDir() + "/tty"
	out, err := os.Create(tty)
	if err != nil {
		t.Fatal(err)
	}

	defer func(action func() error) { _ = action() }(out.Close)

	installer := &installer{
		tty:     out,
		scanner: bufio.NewScanner(strings.NewReader("none\n")),
	}
	interfaces, gateway := installer.askInterfaces([]string{"eth0"})

	iface := interfaces["eth0"]
	if iface == nil || iface.Select != "eth0" {
		t.Fatalf("interface = %#v", iface)
	}

	if len(iface.Addresses) != 0 {
		t.Fatalf("none addressing produced addresses: %#v", iface.Addresses)
	}

	if gateway != "" {
		t.Fatalf("none addressing produced gateway %q", gateway)
	}
}

func TestAskInterfacesUsesSimpleRoles(t *testing.T) {
	tty := t.TempDir() + "/tty"
	out, err := os.Create(tty)
	if err != nil {
		t.Fatal(err)
	}

	defer func(action func() error) { _ = action() }(out.Close)
	installer := &installer{tty: out, scanner: bufio.NewScanner(strings.NewReader("internet\nlocal\n192.168.50.1/24\n"))}
	interfaces, gateway := installer.askInterfaces([]string{"eth0", "eth1"})
	if got := interfaces["eth0"].Addresses; len(got) != 2 || got[0] != "dhcp" || got[1] != "slaac" {
		t.Fatalf("internet addresses = %#v", got)
	}

	if got := interfaces["eth1"].Addresses; len(got) != 1 || got[0] != "192.168.50.1/24" {
		t.Fatalf("local addresses = %#v", got)
	}

	if gateway != "" {
		t.Fatalf("gateway = %q", gateway)
	}
}
