package takeover

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for path, body := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBackupCopiesPresentTargets(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"etc/network/interfaces":       "auto eth0\niface eth0 inet static\n",
		"etc/resolv.conf":              "nameserver 192.0.2.53\n",
		"etc/sysctl.d/10-forward.conf": "net.ipv4.ip_forward=1\n",
	})

	dir := filepath.Join(t.TempDir(), "backup")
	saved, err := Backup(root, dir)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"/etc/network/interfaces", "/etc/resolv.conf", "/etc/sysctl.d"}
	if !slices.Equal(saved, want) {
		t.Fatalf("saved = %v, want %v", saved, want)
	}

	data, err := os.ReadFile(filepath.Join(dir, "etc/sysctl.d/10-forward.conf"))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "net.ipv4.ip_forward=1\n" {
		t.Fatalf("copied file = %q", data)
	}

	manifest, err := os.ReadFile(filepath.Join(dir, "manifest.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(string(manifest)) != strings.Join(want, "\n") {
		t.Fatalf("manifest = %q", manifest)
	}

	fi, err := os.Stat(filepath.Join(dir, "restore.sh"))
	if err != nil {
		t.Fatal(err)
	}

	if fi.Mode().Perm()&0100 == 0 {
		t.Fatalf("restore.sh not executable: %v", fi.Mode())
	}
}

func TestBackupWithNothingToSave(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "backup")
	saved, err := Backup(t.TempDir(), dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(saved) != 0 {
		t.Fatalf("saved = %v, want none", saved)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("backup dir created without targets: %v", err)
	}
}

func TestNativeEnabled(t *testing.T) {
	root := t.TempDir()
	if NativeEnabled(root) {
		t.Fatal("empty root reported native networking")
	}

	if err := os.MkdirAll(filepath.Join(root, "etc/runlevels/boot"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink("/etc/init.d/networking", filepath.Join(root, "etc/runlevels/boot/networking")); err != nil {
		t.Fatal(err)
	}

	if !NativeEnabled(root) {
		t.Fatal("networking in boot runlevel not detected")
	}
}
