package diffutil

import "testing"

func TestLinesAddRemove(t *testing.T) {
	lines := Lines("a\nb\nc\n", "a\nB\nc\n")

	var adds, removes int
	for _, l := range lines {
		switch l.Type {
		case "add":
			if l.Text != "B" {
				t.Fatalf("unexpected add %q", l.Text)
			}

			adds++
		case "remove":
			if l.Text != "b" {
				t.Fatalf("unexpected remove %q", l.Text)
			}

			removes++
		}
	}

	if adds != 1 || removes != 1 {
		t.Fatalf("want 1 add + 1 remove, got %d/%d", adds, removes)
	}
}

func TestFilesStatuses(t *testing.T) {
	before := map[string]string{"same.conf": "x", "mod.conf": "old", "gone.conf": "bye"}
	after := map[string]string{"same.conf": "x", "mod.conf": "new", "new.conf": "hi"}

	got := map[string]string{}
	for _, fd := range Files(before, after) {
		got[fd.File] = fd.Status
	}

	if _, ok := got["same.conf"]; ok {
		t.Fatal("unchanged file should be omitted")
	}

	want := map[string]string{"mod.conf": "modified", "gone.conf": "removed", "new.conf": "added"}
	for f, status := range want {
		if got[f] != status {
			t.Fatalf("%s: want %q, got %q", f, status, got[f])
		}
	}
}
