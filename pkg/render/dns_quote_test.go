package render

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func TestQuoteTXTSplitsLongStrings(t *testing.T) {
	got := quoteTXT(strings.Repeat("a", 300))
	want := `"` + strings.Repeat("a", 255) + `" "` + strings.Repeat("a", 45) + `"`

	if got != want {
		t.Fatalf("quoteTXT = %s", got)
	}
}

func TestQuoteTXTEscapesZoneSyntax(t *testing.T) {
	if got := quoteTXT(`caf` + "é" + ` "x" \`); got != `"caf\195\169 \"x\" \\"` {
		t.Fatalf("quoteTXT = %s", got)
	}

	if got := quoteTXT(`"v=spf1 -all"`); got != `"v=spf1 -all"` {
		t.Fatalf("pre-quoted value changed: %s", got)
	}
}

func TestBindListenFamilies(t *testing.T) {
	cfg := &config.Config{DNS: &config.DNS{Server: &config.DNSServer{Enabled: true, Listen: []string{"0.0.0.0"}}}}

	if got := bindListen4(cfg); got != "any;" {
		t.Fatalf("bindListen4 = %q", got)
	}

	if got := bindListen6(cfg); got != "" {
		t.Fatalf("bindListen6 = %q, want no IPv6 listener", got)
	}
}
