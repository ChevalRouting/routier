package switchcache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppliedSinceFollowsTheArtifact(t *testing.T) {
	dir := t.TempDir()
	previous := artifactDir
	artifactDir = dir + "/"
	t.Cleanup(func() { artifactDir = previous })

	if appliedSince("csw1", time.Time{}) {
		t.Fatal("a switch without an artifact is reported as applied")
	}

	path := filepath.Join(dir, "csw1.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	read := time.Now()
	if err := os.Chtimes(path, read.Add(-time.Minute), read.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	if appliedSince("csw1", read) {
		t.Fatal("an artifact written before the read makes the entry stale")
	}

	if err := os.Chtimes(path, read.Add(time.Second), read.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	if !appliedSince("csw1", read) {
		t.Fatal("an artifact written after the read keeps the entry fresh")
	}
}
