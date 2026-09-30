package config

import (
	"testing"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestParseNftErrors(t *testing.T) {
	out := `/tmp/routier-nftcheck-123.nft:12:7-9: Error: syntax error, unexpected newline
/tmp/routier-nftcheck-123.nft:14:1-3: Error: unknown chain bogus`

	errs := parseNftErrors(out)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %+v", len(errs), errs)
	}

	if errs[0].Line != 12 || errs[0].Message != "syntax error, unexpected newline" {
		t.Fatalf("unexpected first error: %+v", errs[0])
	}

	if errs[1].Line != 14 {
		t.Fatalf("unexpected second error line: %+v", errs[1])
	}
}

func TestParseNftErrorsFallback(t *testing.T) {
	errs := parseNftErrors("nft: command not found")
	if len(errs) != 1 || errs[0].Line != 0 || errs[0].Message != "nft: command not found" {
		t.Fatalf("expected single fallback error, got: %+v", errs)
	}
}

func TestLocateNftErrorsChainLine(t *testing.T) {
	ruleset := `#!/usr/sbin/nft -f
flush ruleset

table inet routier {
	chain input {
		type filter hook input priority filter; policy drop;
		ip saddr 10.0.0.0/8 accept
		tcp dport bogus accept
	}
}
`
	draft := cfgpkg.NftablesConfig{
		Chains: map[string]*cfgpkg.NftChain{
			"input": {Rules: "ip saddr 10.0.0.0/8 accept\ntcp dport bogus accept"},
		},
	}

	errs := []types.NftValidateError{{Line: 8, Message: "syntax error"}}
	locateNftErrors(errs, ruleset, draft)

	if errs[0].Section != "input" || errs[0].Line != 2 {
		t.Fatalf("expected input line 2, got: %+v", errs[0])
	}
}
