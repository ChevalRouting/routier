package apply

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadSnapshotArtifactUsesSavedContent(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "snapshots")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	live := filepath.Join(root, "live.conf")
	if err := os.WriteFile(live, []byte("current content"), 0o600); err != nil {
		t.Fatal(err)
	}

	saved := snapshot{ID: "test-id", Files: map[string]string{live: "previous content", "/empty.conf": ""}}
	data, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "test-id.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	content, err := readSnapshotArtifact(root, "test-id", live)
	if err != nil || content != "previous content" {
		t.Fatalf("saved artifact = %q, %v", content, err)
	}

	content, err = readSnapshotArtifact(root, "test-id", "/empty.conf")
	if err != nil || content != "" {
		t.Fatalf("empty artifact = %q, %v", content, err)
	}

	if _, err := readSnapshotArtifact(root, "test-id", "/etc/passwd"); err == nil {
		t.Fatal("accepted a path outside the snapshot")
	}

	for _, id := range []string{"", "../test-id", "/test-id", `..\test-id`, "missing"} {
		if _, err := readSnapshotArtifact(root, id, live); err == nil {
			t.Fatalf("accepted snapshot id %q", id)
		}
	}
}
