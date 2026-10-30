package apitest

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/tests/testkit"
)

func TestConfirmPreservesNewerDraft(t *testing.T) {
	t.Cleanup(managers.SetPendingFile(filepath.Join(t.TempDir(), "pending")))
	node := testkit.NewNode(t, testkit.MinimalConfig)
	newer := []byte(strings.ReplaceAll(testkit.MinimalConfig, "rtr1", "next-draft"))
	path := cfgstore.StagingPath(node.Cfg, "admin")
	if err := os.WriteFile(path, newer, 0600); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		code, body := node.Do(t, http.MethodPost, "/api/apply/confirm", nil)
		if code != http.StatusOK {
			t.Fatalf("confirm: %d %s", code, body)
		}

		staged, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(staged, newer) {
			t.Fatalf("confirmation replaced newer draft: %s", staged)
		}
	}
}

func TestConfirmDiscardsOnlyAppliedDraft(t *testing.T) {
	t.Cleanup(managers.SetPendingFile(filepath.Join(t.TempDir(), "pending")))
	node := testkit.NewNode(t, testkit.MinimalConfig)
	for _, username := range []string{"admin", "other"} {
		if err := os.WriteFile(cfgstore.StagingPath(node.Cfg, username), []byte(testkit.MinimalConfig), 0600); err != nil {
			t.Fatal(err)
		}
	}

	for range 2 {
		code, body := node.Do(t, http.MethodPost, "/api/apply/confirm", nil)
		if code != http.StatusOK {
			t.Fatalf("confirm: %d %s", code, body)
		}
	}

	if _, err := os.Stat(cfgstore.StagingPath(node.Cfg, "admin")); !os.IsNotExist(err) {
		t.Fatalf("applied draft remains: %v", err)
	}

	if _, err := os.Stat(cfgstore.StagingPath(node.Cfg, "other")); err != nil {
		t.Fatalf("another user's draft was removed: %v", err)
	}
}
