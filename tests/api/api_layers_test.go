package apitest

import (
	"net/http"
	"testing"

	"github.com/ChevalRouting/routier/tests/testkit"
)

func TestAPIProjectsSimpleLayer(t *testing.T) {
	n := testkit.NewNode(t, testkit.FullConfig)

	status, raw := n.Do(t, http.MethodGet, "/api/config?layer=simple", nil)
	if status != http.StatusOK {
		t.Fatalf("get simple config %d: %s", status, raw)
	}

	projection := testkit.DecodeData[map[string]any](t, raw)
	if projection["layer"] != "simple" {
		t.Fatalf("layer = %v", projection["layer"])
	}
	if _, ok := projection["sections"].(map[string]any); !ok {
		t.Fatalf("sections = %#v", projection["sections"])
	}
	if _, ok := projection["losses"].([]any); !ok {
		t.Fatalf("losses = %#v", projection["losses"])
	}
}

func TestAPIRestrictsStagingToOneLayer(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	status, raw := n.Do(t, http.MethodPut, "/api/config/system?layer=simple", map[string]any{
		"hostname": "simple-router",
	})
	if status != http.StatusOK {
		t.Fatalf("put simple system %d: %s", status, raw)
	}

	status, _ = n.Do(t, http.MethodPut, "/api/config/hostname?layer=advanced", "advanced-router")
	if status != http.StatusConflict {
		t.Fatalf("advanced write status = %d, want %d", status, http.StatusConflict)
	}

	status, _ = n.Do(t, http.MethodPost, "/api/config/apply?layer=advanced", nil)
	if status != http.StatusConflict {
		t.Fatalf("advanced apply status = %d, want %d", status, http.StatusConflict)
	}

	status, _ = n.Do(t, http.MethodDelete, "/api/config/staging?layer=advanced", nil)
	if status != http.StatusConflict {
		t.Fatalf("advanced discard status = %d, want %d", status, http.StatusConflict)
	}

	status, raw = n.Do(t, http.MethodDelete, "/api/config/staging?layer=simple", nil)
	if status != http.StatusOK {
		t.Fatalf("simple discard %d: %s", status, raw)
	}
}

func TestAPIReadsCommittedConfigOutsideStagingLayer(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	status, raw := n.Do(t, http.MethodPut, "/api/config/system?layer=simple", map[string]any{
		"hostname": "simple-router",
	})
	if status != http.StatusOK {
		t.Fatalf("put simple system %d: %s", status, raw)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/config/hostname?layer=advanced", nil)
	if status != http.StatusOK {
		t.Fatalf("get advanced hostname %d: %s", status, raw)
	}
	if hostname := testkit.DecodeData[string](t, raw); hostname != "rtr1" {
		t.Fatalf("advanced hostname = %q, want committed hostname", hostname)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/config/system?layer=simple", nil)
	if status != http.StatusOK {
		t.Fatalf("get simple system %d: %s", status, raw)
	}
	system := testkit.DecodeData[map[string]any](t, raw)
	if system["hostname"] != "simple-router" {
		t.Fatalf("simple hostname = %v", system["hostname"])
	}
}
