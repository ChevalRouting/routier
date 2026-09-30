package mcptest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/routier/pkg/mcpserver"
)

func TestServerLoadsMultipleInstances(t *testing.T) {
	t.Setenv("ROUTIER_MCP_TEST_B", "token-b")

	path := filepath.Join(t.TempDir(), "instances.yml")
	data := []byte(`instances:
  edge-a:
    url: https://192.0.2.1:8080
    token: token-a
  edge-b:
    url: https://192.0.2.2:8080
    token_env: ROUTIER_MCP_TEST_B
    tls:
      skip_verify: true
`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := mcpserver.New(path, "test"); err != nil {
		t.Fatalf("new server: %v", err)
	}
}

func TestServerRejectsTokenAndTokenEnv(t *testing.T) {
	t.Setenv("ROUTIER_MCP_TEST_DUPLICATE", "token-env")

	path := filepath.Join(t.TempDir(), "instances.yml")
	data := []byte(`instances:
  edge-a:
    url: https://192.0.2.1:8080
    token: token-file
    token_env: ROUTIER_MCP_TEST_DUPLICATE
`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := mcpserver.New(path, "test"); err == nil {
		t.Fatal("expected ambiguous token error")
	}
}

func TestServerRejectsMissingToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "instances.yml")
	data := []byte(`instances:
  edge-a:
    url: https://192.0.2.1:8080
    token_env: ROUTIER_MCP_TEST_MISSING
`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := mcpserver.New(path, "test"); err == nil {
		t.Fatal("expected missing token error")
	}
}
