//go:build linux

package netlink

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func fakeSysctls(t *testing.T, keys []string, value string) string {
	root := t.TempDir()
	for _, key := range keys {
		path := filepath.Join(root, strings.ReplaceAll(key, ".", "/"))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(value+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func bridgeSysctls(t *testing.T) string {
	return fakeSysctls(t, bridgeNetfilterKeys, "1")
}

func sysctlValue(t *testing.T, root, key string) string {
	data, err := os.ReadFile(filepath.Join(root, strings.ReplaceAll(key, ".", "/")))
	if err != nil {
		t.Fatal(err)
	}

	return strings.TrimSpace(string(data))
}

func TestBridgedFramesLeaveNetfilter(t *testing.T) {
	dir := bridgeSysctls(t)
	cfg := &config.Config{
		Interfaces: map[string]*config.Interface{"fabric": {Type: "vxlan-bridge"}},
		Sysctl:     map[string]string{"net.bridge.bridge-nf-call-arptables": "1"},
	}

	keepBridgedFramesOutOfNetfilter(cfg, dir, false)
	if sysctlValue(t, dir, "net.bridge.bridge-nf-call-iptables") != "0" || sysctlValue(t, dir, "net.bridge.bridge-nf-call-ip6tables") != "0" {
		t.Fatal("bridged frames must not go through iptables")
	}

	if sysctlValue(t, dir, "net.bridge.bridge-nf-call-arptables") != "1" {
		t.Fatal("a value set in sysctl is kept")
	}
}

func TestNoBridgeLeavesNetfilterAlone(t *testing.T) {
	dir := bridgeSysctls(t)
	keepBridgedFramesOutOfNetfilter(&config.Config{Interfaces: map[string]*config.Interface{"lan": {Select: "eth0"}}}, dir, false)
	if sysctlValue(t, dir, "net.bridge.bridge-nf-call-iptables") != "1" {
		t.Fatal("without a bridge nothing changes")
	}
}

func TestMissingBridgeModule(t *testing.T) {
	cfg := &config.Config{Interfaces: map[string]*config.Interface{"br0": {Type: "bridge"}}}
	keepBridgedFramesOutOfNetfilter(cfg, filepath.Join(t.TempDir(), "absent"), false)
}
