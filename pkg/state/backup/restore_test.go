package backup

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type archiveEntry struct {
	name string
	data string
}

func TestRestoreKeepsFilesInsideTheConfigDirectory(t *testing.T) {
	if _, err := exec.LookPath("zstd"); err != nil {
		t.Skip("zstd not installed")
	}

	root := t.TempDir()
	outside := t.TempDir()
	configPath := filepath.Join(root, "routier.yml")
	template := filepath.Join(root, "templates", "motd.tmpl")
	escape := filepath.Join(outside, "doas.conf")
	manifestJSON, err := json.Marshal(manifest{Version: "1", ConfigPath: "/etc/routier/routier.yml"})
	if err != nil {
		t.Fatal(err)
	}

	archive := writeArchive(t, []archiveEntry{
		{name: "manifest.json", data: string(manifestJSON)},
		{name: "etc/routier/routier.yml", data: "hostname: restored\n"},
		{name: template[1:], data: "hello"},
		{name: escape[1:], data: "permit nopass nobody"},
		{name: "../../" + escape[1:], data: "permit nopass nobody"},
	})

	restored, err := Restore(archive, configPath)
	if err != nil {
		t.Fatal(err)
	}

	if restored.ConfigPath != configPath || string(restored.Config) != "hostname: restored\n" {
		t.Fatalf("restored %q with %q", restored.ConfigPath, restored.Config)
	}

	if data, err := os.ReadFile(template); err != nil || string(data) != "hello" {
		t.Fatalf("template: %q, %v", data, err)
	}

	if _, err := os.Stat(escape); !os.IsNotExist(err) {
		t.Fatalf("file outside the config directory was written: %v", err)
	}
}

func writeArchive(t *testing.T, entries []archiveEntry) string {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: e.name, Mode: 0644, Size: int64(len(e.data))}); err != nil {
			t.Fatal(err)
		}

		if _, err := tw.Write([]byte(e.data)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "backup.tar.zst")
	cmd := exec.Command("zstd", "-q", "-o", path)
	cmd.Stdin = &buf
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("zstd: %v: %s", err, out)
	}

	return path
}
