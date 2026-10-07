package apitest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/ChevalRouting/routier/tests/testkit"
)

func apiKeyRequest(t *testing.T, token, method, url string, body any) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}

		reader = bytes.NewReader(data)
	}

	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}

	defer func(action func() error) { _ = action() }(response.Body.Close)

	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	return response.StatusCode, data
}

func TestApplicationAPIKeyLifecycle(t *testing.T) {
	node := testkit.NewNode(t, testkit.MinimalConfig)
	status, body := node.Do(t, http.MethodPost, "/api/auth/api-keys", types.CreateAPIKeyRequest{Name: "automation"})
	if status != http.StatusOK {
		t.Fatalf("create API key: %d: %s", status, body)
	}

	created := testkit.DecodeData[types.CreateAPIKeyResponse](t, body)
	if created.Token == "" || created.APIKey.Name != "automation" || created.APIKey.Prefix == "" {
		t.Fatalf("invalid API key response: %+v", created)
	}

	status, _ = apiKeyRequest(t, created.Token, http.MethodGet, node.URL+"/api/config", nil)
	if status != http.StatusOK {
		t.Fatalf("API key request returned %d", status)
	}

	status, _ = apiKeyRequest(t, created.Token, http.MethodGet, node.URL+"/api/auth/api-keys", nil)
	if status != http.StatusForbidden {
		t.Fatalf("API key managed credentials with status %d", status)
	}

	status, body = node.Do(t, http.MethodGet, "/api/auth/api-keys", nil)
	if status != http.StatusOK {
		t.Fatalf("list API keys: %d: %s", status, body)
	}

	keys := testkit.DecodeData[[]types.APIKey](t, body)
	if len(keys) != 1 || keys[0].ID != created.APIKey.ID {
		t.Fatalf("listed API keys: %+v", keys)
	}

	if bytes.Contains(body, []byte(created.Token)) {
		t.Fatal("API key token returned by list endpoint")
	}

	status, body = node.Do(t, http.MethodDelete, "/api/auth/api-keys/"+created.APIKey.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("revoke API key: %d: %s", status, body)
	}

	status, _ = apiKeyRequest(t, created.Token, http.MethodGet, node.URL+"/api/config", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("revoked API key returned %d", status)
	}
}
