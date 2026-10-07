package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestTextRules(t *testing.T) {
	if issues := checkText("docs.md", []byte("Use a comma, colon or parentheses.\n")); len(issues) != 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}

	issues := checkText("docs.md", []byte("First line\nReplace this \u2014 punctuation.\n"))
	if len(issues) != 1 || issues[0].line != 2 || issues[0].rule != "text" {
		t.Fatalf("incorrect text diagnostics: %+v", issues)
	}
}

func TestSourcePathsRespectGitIgnores(t *testing.T) {
	root := t.TempDir()
	command := exec.Command("git", "init", "-q", root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("initialize repository: %v: %s", err, output)
	}

	files := map[string]string{
		".gitignore":  "ignored.txt\n",
		"tracked.md":  "Tracked text\n",
		"new.md":      "New text\n",
		"ignored.txt": "Ignored text\n",
	}
	for path, data := range files {
		if err := os.WriteFile(filepath.Join(root, path), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	command = exec.Command("git", "-C", root, "add", "tracked.md", ".gitignore")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("stage fixture: %v: %s", err, output)
	}

	paths, err := sourcePaths(options{root: root})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(paths, "tracked.md") || !slices.Contains(paths, "new.md") || slices.Contains(paths, "ignored.txt") {
		t.Fatalf("incorrect source paths: %v", paths)
	}
}

func TestCheckFilesExcludesGeneratedAndBinaryFiles(t *testing.T) {
	root := t.TempDir()
	generated := filepath.Join(root, "pkg", "db", "generated")
	if err := os.MkdirAll(generated, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(generated, "query.go"), []byte("Generated \u2014 text"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(root, "binary.dat"), []byte{0xff, 0x00, 0xa0}, 0o600); err != nil {
		t.Fatal(err)
	}

	if !checkFiles(options{root: root, only: "style"}, []string{"pkg/db/generated/query.go", "binary.dat"}) {
		t.Fatal("generated or binary content was linted")
	}
}

func TestCheckFilesReportsTextFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "docs.md"), []byte("Invalid \u2014 punctuation"), 0o600); err != nil {
		t.Fatal(err)
	}

	if checkFiles(options{root: root, only: "text"}, []string{"docs.md"}) {
		t.Fatal("text violation did not fail validation")
	}
}

func TestExplicitMissingFileFails(t *testing.T) {
	settings := options{root: t.TempDir(), only: "style", paths: []string{"missing.go"}}
	if checkFiles(settings, settings.paths) {
		t.Fatal("explicit missing file was silently skipped")
	}
}
