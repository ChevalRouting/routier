package apitest

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/tests/testkit"
)

type v1ConfigDocument struct {
	YAML   string `json:"yaml"`
	SHA256 string `json:"sha256"`
}

type v1SessionInfo struct {
	ID string `json:"id"`
}

func TestV1CanonicalConfigSessionYAML(t *testing.T) {
	n := testkit.NewNode(t, testkit.MinimalConfig)

	status, raw := n.Do(t, http.MethodGet, "/api/v1/config", nil)
	if status != http.StatusOK {
		t.Fatalf("get config %d: %s", status, raw)
	}

	doc := testkit.DecodeData[v1ConfigDocument](t, raw)
	if !strings.Contains(doc.YAML, "hostname: rtr1") || len(doc.SHA256) != 64 {
		t.Fatalf("unexpected config document: %+v", doc)
	}

	status, raw = n.Do(t, http.MethodPost, "/api/v1/sessions", nil)
	if status != http.StatusOK {
		t.Fatalf("create session %d: %s", status, raw)
	}

	session := testkit.DecodeData[v1SessionInfo](t, raw)
	yaml := strings.Replace(doc.YAML, "hostname: rtr1", "hostname: mcp-router", 1)
	req, err := http.NewRequest(http.MethodPut, n.URL+"/api/v1/sessions/"+session.ID+"/config", strings.NewReader(yaml))
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+n.JWT)
	req.Header.Set("Content-Type", "application/yaml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put session config: %v", err)
	}

	defer func(action func() error) { _ = action() }(resp.Body.Close)
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put session config %d: %s", resp.StatusCode, body)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/v1/sessions/"+session.ID+"/config", nil)
	if status != http.StatusOK {
		t.Fatalf("get session config %d: %s", status, raw)
	}

	updated := testkit.DecodeData[v1ConfigDocument](t, raw)
	if !strings.Contains(updated.YAML, "hostname: mcp-router") {
		t.Fatalf("session YAML was not replaced: %s", updated.YAML)
	}

	status, raw = n.Do(t, http.MethodGet, "/api/v1/config", nil)
	committed := testkit.DecodeData[v1ConfigDocument](t, raw)
	if status != http.StatusOK || strings.Contains(committed.YAML, "mcp-router") {
		t.Fatalf("session write changed committed config: %s", committed.YAML)
	}
}
