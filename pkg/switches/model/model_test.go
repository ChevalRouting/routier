package model

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExpandPorts(t *testing.T) {
	ports, err := ExpandPorts("ether[1-3], sfp-sfpplus[1-2],qsfp28-1-1")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"ether1", "ether2", "ether3", "sfp-sfpplus1", "sfp-sfpplus2", "qsfp28-1-1"}
	if !slices.Equal(ports, want) {
		t.Fatalf("got %v", ports)
	}

	if _, err := ExpandPorts("ether[3-1]"); err == nil {
		t.Fatal("reversed range accepted")
	}
}

func TestMatchPrefersSpecificModels(t *testing.T) {
	catalog, err := Load("")
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"MikroTik RouterOS 7.16 (stable) CRS310-8G+2S+":   "mikrotik-crs310-8g-2s",
		"MikroTik RouterOS 7.16.1 (stable) CRS309-1G-8S+": "mikrotik-crs309-1g-8s",
		"MikroTik RouterOS 7.15 (stable) CRS326-24G-2S+":  "mikrotik-generic",
	}
	for sysDescr, want := range cases {
		model, ok := catalog.Match(sysDescr)
		if !ok || model.ID != want {
			t.Errorf("%s: matched %q", sysDescr, model.ID)
		}
	}

	if _, ok := catalog.Match("Cisco IOS Software, C2960"); ok {
		t.Error("matched an unknown vendor")
	}
}

func TestBuiltinFaceplates(t *testing.T) {
	catalog, err := Load("")
	if err != nil {
		t.Fatal(err)
	}

	crs310, err := catalog.Get("mikrotik-crs310-8g-2s")
	if err != nil {
		t.Fatal(err)
	}

	if len(crs310.Faceplate) != 2 || len(crs310.Faceplate[0].Ports) != 8 || crs310.Faceplate[1].Ports[1] != "sfp-sfpplus2" {
		t.Fatalf("unexpected faceplate %+v", crs310.Faceplate)
	}

	crs309, err := catalog.Get("mikrotik-crs309-1g-8s")
	if err != nil {
		t.Fatal(err)
	}

	if len(crs309.Faceplate) != 2 || len(crs309.Faceplate[0].Ports) != 8 || crs309.Faceplate[1].Ports[0] != "ether1" {
		t.Fatalf("unexpected faceplate %+v", crs309.Faceplate)
	}

	if len(crs309.Credentials) != 2 || !crs309.Credentials[1].Required {
		t.Fatalf("unexpected credentials %+v", crs309.Credentials)
	}
}

func TestExtraDirOverridesBuiltin(t *testing.T) {
	dir := t.TempDir()
	custom := "id: mikrotik-crs310-8g-2s\nvendor: mikrotik\nname: Custom\ndriver: routeros-rest\nmatch:\n  sys_descr: 'CRS310'\n"
	if err := os.WriteFile(filepath.Join(dir, "crs310.yaml"), []byte(custom), 0o600); err != nil {
		t.Fatal(err)
	}

	catalog, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	model, err := catalog.Get("mikrotik-crs310-8g-2s")
	if err != nil || model.Name != "Custom" {
		t.Fatalf("override not applied: %+v %v", model, err)
	}
}
