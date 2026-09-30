package apitest

import (
	"testing"

	"github.com/ChevalRouting/routier/pkg/diffutil"
)

func TestConfigDiffIgnoresFileFormatting(t *testing.T) {
	before := []byte(`# edited by hand
hostname: old
interfaces:
  core:
    type: 'bond'
    bond: {mode: 802.3ad, members: [core1, core2]}
`)
	after := []byte(`interfaces:
    core:
        bond:
            members:
                - core1
                - core2
            mode: 802.3ad
        type: bond
hostname: new
`)
	lines, err := diffutil.YAML(before, after)
	if err != nil {
		t.Fatal(err)
	}

	changed := 0
	for _, line := range lines {
		if line.Type != "same" {
			changed++
		}
	}

	if changed != 2 {
		t.Fatalf("expected only hostname removal/addition, got %d: %+v", changed, lines)
	}

	lines, err = diffutil.YAML(before, before)
	if err != nil {
		t.Fatal(err)
	}

	for _, line := range lines {
		if line.Type != "same" {
			t.Fatalf("unchanged config: %+v", lines)
		}
	}

	if _, err = diffutil.YAML([]byte("interfaces: ["), after); err == nil {
		t.Fatal("invalid YAML accepted")
	}
}
