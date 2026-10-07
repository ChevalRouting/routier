package render

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

type nftNameCase struct {
	description string
	name        string
}

func TestBGPDescriptionNftVariablesAreSafe(t *testing.T) {
	cases := []nftNameCase{
		{"Transit provider (AS64500)", "Transit_provider__AS64500_"},
		{"peer/eu-west:1.example", "peer_eu_west_1_example"},
		{"peer\tprimary\nbackup", "peer_primary_backup"},
		{"peer\"; define injected = 1 #", "peer___define_injected___1__"},
		{"東京 peer 🌍", "___peer__"},
		{"existing_peer_123", "existing_peer_123"},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) { testBGPDescriptionNftVariables(t, tc) })
	}
}

func TestNftAddressNamesRemainCompatible(t *testing.T) {
	if got := sanitizeNftName("2001:db8::1"); got != "2001_db8__1" {
		t.Fatalf("IPv6 name changed: %q", got)
	}
}

func testBGPDescriptionNftVariables(t *testing.T, tc nftNameCase) {
	t.Helper()
	validName := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	cfg := &config.Config{Routing: &config.Routing{BGP: &config.BGP{Neighbors: []config.BGPNeighbor{{Address: "192.0.2.1", Description: tc.description}}}}}
	vars := NftVars(cfg)
	found, addressFound := false, false
	for _, variable := range vars {
		if !validName.MatchString(variable.Name) {
			t.Fatalf("invalid rendered variable: %q", variable.Name)
		}

		if variable.Name == "bgp_neighbor_"+tc.name {
			found = true
		}

		if variable.Name == "bgp_neighbor_192_0_2_1" {
			addressFound = true
		}

		if variable.Name == "injected" {
			t.Fatal("description injected a define")
		}
	}

	if !found || !addressFound {
		t.Fatalf("missing description or address variable: %+v", vars)
	}

	raw := templateFuncs(TemplateData{Config: cfg})["nftDefines"].(func() string)()
	if !strings.Contains(raw, "define bgp_neighbor_"+tc.name+" = 192.0.2.1\n") {
		t.Fatalf("unsafe or inconsistent ruleset defines:\n%s", raw)
	}

	if cfg.Routing.BGP.Neighbors[0].Description != tc.description {
		t.Fatal("changed the BGP description itself")
	}
}
