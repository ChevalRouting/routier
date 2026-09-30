package updates

import "testing"

func TestParseUpgradable(t *testing.T) {
	out := `Installed:                Available:
busybox-1.36.1-r5 < 1.36.1-r15
routier-ui-0.3.0-r0 < 0.4.0-r0
linux-lts-6.12.4-r0 < 6.12.11-r0
`

	pkgs := parseUpgradable(out)
	if len(pkgs) != 3 {
		t.Fatalf("got %d packages: %+v", len(pkgs), pkgs)
	}

	want := []Package{
		{Name: "busybox", Old: "1.36.1-r5", New: "1.36.1-r15"},
		{Name: "routier-ui", Old: "0.3.0-r0", New: "0.4.0-r0"},
		{Name: "linux-lts", Old: "6.12.4-r0", New: "6.12.11-r0"},
	}

	for i, w := range want {
		if pkgs[i] != w {
			t.Fatalf("package %d = %+v, want %+v", i, pkgs[i], w)
		}
	}
}

func TestParseUpgradableSkipsNoise(t *testing.T) {
	out := "WARNING: opening from cache: no such file\n\nfoo-bar-2.0-r1 < 2.1-r0\ngarbage line\n"

	pkgs := parseUpgradable(out)
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages: %+v", len(pkgs), pkgs)
	}

	if pkgs[0].Name != "foo-bar" || pkgs[0].Old != "2.0-r1" || pkgs[0].New != "2.1-r0" {
		t.Fatalf("unexpected package: %+v", pkgs[0])
	}
}

func TestContainsHelpers(t *testing.T) {
	pkgs := []Package{{Name: "routier-ui"}, {Name: "openssl"}}
	if !containsAny(pkgs, "routier", "routier-ui") {
		t.Fatal("expected self-update match")
	}

	if containsAny(pkgs, "frr") {
		t.Fatal("unexpected match")
	}
}
