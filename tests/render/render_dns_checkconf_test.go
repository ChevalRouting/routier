package rendertest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderNamedConfPassesCheckconf(t *testing.T) {
	if _, err := exec.LookPath("named-checkconf"); err != nil {
		t.Skip("named-checkconf not installed")
	}

	cases := map[string]string{
		"empty primary":     dnsEmptyPrimaryConfig,
		"target":            dnsTargetConfig,
		"transfers":         strings.Replace(dnsZoneConfig, "        ttl: 300", "        notify_source: 198.51.100.11\n        notify_source_v6: fd00:42::1\n        secondaries: [192.0.2.2, 192.0.2.3]\n        transfer_key: {name: transfer.home.arpa, secret: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=}\n        ttl: 300", 1),
		"zones":             dnsZoneConfig,
		"forwarder":         dnsForwarderOnly,
		"authoritative":     dnsAuthoritativeOnly,
		"both":              dnsBothRoles,
		"views":             dnsViewsConfig,
		"view sources":      strings.Replace(dnsViewsConfig, "          - name: corp.example\n", "          - name: corp.example\n            notify_source: 198.51.100.11\n            notify_source_v6: fd00:42::1\n", 1),
		"secondary sources": dnsServerBaseSourceConfig,
		"automatic sources": strings.Replace(dnsZoneConfig, "        ttl: 300", "        notify_source: '*'\n        notify_source_v6: '*'\n        ttl: 300", 1),
	}

	for name, yaml := range cases {
		t.Run(name, func(t *testing.T) { checkNamedConfPasses(yaml, t) })
	}
}

func checkNamedConfPasses(yaml string, t *testing.T) {
	dir := t.TempDir()
	zoneDir := filepath.Join(dir, "zones")
	if err := os.MkdirAll(zoneDir, 0o755); err != nil {
		t.Fatal(err)
	}

	conf := ""
	for dest, content := range renderByDest(t, yaml) {
		switch {
		case dest == "/etc/bind/named.conf":
			conf = content
		case strings.HasPrefix(dest, "/etc/bind/zones/"):
			path := filepath.Join(zoneDir, filepath.Base(dest))
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	if conf == "" {
		t.Fatal("no named.conf rendered")
	}

	key := filepath.Join(dir, "rndc.key")
	if err := os.WriteFile(key, []byte("key \"rndc-key\" {\n\talgorithm hmac-sha256;\n\tsecret \"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\";\n};\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	staged := strings.ReplaceAll(conf, "/etc/bind/rndc.key", key)
	staged = strings.ReplaceAll(staged, "/etc/bind/zones/", zoneDir+"/")

	path := filepath.Join(dir, "named.conf")
	if err := os.WriteFile(path, []byte(staged), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("named-checkconf", path).CombinedOutput()
	if err != nil {
		t.Fatalf("named-checkconf rejected the rendered config: %v\n%s\n--- config ---\n%s",
			err, out, staged)
	}

	if text := strings.TrimSpace(string(out)); text != "" {
		t.Errorf("named-checkconf reported:\n%s", text)
	}
}

const dnsServerBaseSourceConfig = `version: v3.0.0
hostname: gw
dns:
  server:
    enabled: true
    listen: [127.0.0.1]
    allow_from: [127.0.0.1]
    zones:
      - name: example.net
        primaries: [192.0.2.1]
        transfer_source: 198.51.100.11
        transfer_source_v6: fd00:42::1
`
