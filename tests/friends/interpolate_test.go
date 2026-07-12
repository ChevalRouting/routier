package friendstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
)

func vars() map[string]friends.Vars {
	return map[string]friends.Vars{
		"edge-b": {
			Hostname:    "edge-b",
			Fingerprint: "SHA256:abc",
			Version:     "1.2.3",
			Interfaces: map[string][]string{
				"wan": {"198.51.100.2"},
				"lan": {"10.0.0.2", "10.0.1.2"},
			},
		},
	}
}

func TestInterpolateResolvesPaths(t *testing.T) {
	cases := map[string]string{
		`{{ friend "edge-b" "hostname" }}`:                 "edge-b",
		`{{ friend "edge-b" "fingerprint" }}`:              "SHA256:abc",
		`{{ friend "edge-b" "version" }}`:                  "1.2.3",
		`{{ friend "edge-b" "interfaces.wan.address" }}`:   "198.51.100.2",
		`{{ friend "edge-b" "interfaces.lan.addresses" }}`: "10.0.0.2,10.0.1.2",
	}

	for expr, want := range cases {
		got, err := friends.Interpolate(expr, vars())
		if err != nil || got != want {
			t.Fatalf("%s -> %q, %v (want %q)", expr, got, err, want)
		}
	}
}

func TestInterpolateErrors(t *testing.T) {
	if _, err := friends.Interpolate(`{{ friend "ghost" "hostname" }}`, vars()); err == nil {
		t.Fatal("expected unknown-friend error")
	}

	if _, err := friends.Interpolate(`{{ friend "edge-b" "bogus" }}`, vars()); err == nil {
		t.Fatal("expected unknown-path error")
	}

	if _, err := friends.Interpolate(`{{ friend "edge-b" "interfaces.eth9.address" }}`, vars()); err == nil {
		t.Fatal("expected no-address error")
	}
}

func TestInterpolatePassthrough(t *testing.T) {
	raw := "version: v3.0.0\nhostname: rtr\n"
	got, err := friends.Interpolate(raw, vars())
	if err != nil || got != raw {
		t.Fatalf("passthrough changed config: %q, %v", got, err)
	}
}

func TestLoadCacheVars(t *testing.T) {
	path := filepath.Join(t.TempDir(), "friends_cache.json")
	const cache = `{"status":{"edge-b":{"name":"edge-b","hostname":"edge-b","version":"9.9"}},` +
		`"interfaces":{"edge-b":[{"name":"wan","addresses":["198.51.100.2"]}]}}`
	if err := os.WriteFile(path, []byte(cache), 0600); err != nil {
		t.Fatal(err)
	}

	v := friends.LoadCacheVars(path)
	if v["edge-b"].Hostname != "edge-b" || v["edge-b"].Version != "9.9" {
		t.Fatalf("vars = %+v", v["edge-b"])
	}

	if got := v["edge-b"].Interfaces["wan"]; len(got) != 1 || got[0] != "198.51.100.2" {
		t.Fatalf("wan addresses = %v", got)
	}

	if len(friends.LoadCacheVars(filepath.Join(t.TempDir(), "none.json"))) != 0 {
		t.Fatal("expected empty vars for missing cache")
	}
}

func TestInterpolateConfigRoundTrip(t *testing.T) {
	raw := `version: v3.0.0
hostname: gw
routing:
  static:
    - destination: 10.9.0.0/24
      via: {{ friend "edge-b" "interfaces.wan.address" }}
`
	resolved, err := friends.Interpolate(raw, vars())
	if err != nil {
		t.Fatalf("interpolate: %v", err)
	}

	if !strings.Contains(resolved, "via: 198.51.100.2") {
		t.Fatalf("unresolved:\n%s", resolved)
	}

	cfg, err := config.LoadBytes([]byte(resolved))
	if err != nil {
		t.Fatalf("load resolved: %v", err)
	}

	if cfg.Routing.Static[0].Via != "198.51.100.2" {
		t.Fatalf("via = %q", cfg.Routing.Static[0].Via)
	}
}
