package rendertest

import (
	"github.com/ChevalRouting/routier/tests/harness"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
)

func TestRenderProducesNftablesTable(t *testing.T) {
	dir := t.TempDir()
	path := harness.WriteConfig(t, dir, `version: v3.0.0
hostname: rtr
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.1/24"]
nftables:
  chains:
    input:
      rules: |
        tcp dport 22 accept
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	outputs, err := render.All(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if len(outputs) == 0 {
		t.Fatal("no render outputs")
	}

	var nft string
	for _, o := range outputs {
		if strings.Contains(o.Content, "table inet routier") {
			nft = o.Content
		}
	}

	if nft == "" {
		t.Fatal("rendered output missing routier nftables table")
	}

	if !strings.Contains(nft, "dport 22 accept") {
		t.Fatalf("user rule not rendered into nftables:\n%s", nft)
	}
}

func TestRenderNftVarsExposesInterfaceAddresses(t *testing.T) {
	dir := t.TempDir()
	path := harness.WriteConfig(t, dir, `version: v3.0.0
hostname: rtr
interfaces:
  lan:
    select: name=eth0
    addresses: ["10.0.0.1/24"]
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	vars := render.NftVars(cfg)
	found := false
	for _, v := range vars {
		if v.Name == "lan_network" || v.Name == "lan_address" {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected lan_* nft vars, got %+v", vars)
	}
}
