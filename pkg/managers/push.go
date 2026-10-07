package managers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
)

const identityKeyPath = "/var/lib/routier/identity_ed25519"

func sealForFriend(f *config.Friend, data []byte) ([]byte, error) {
	if f.Identity.X25519PublicKey == "" {
		return nil, fmt.Errorf("friend %q has no encryption key; re-pair the friend", f.Name)
	}

	id, err := identity.LoadOrCreate(identityKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load identity: %w", err)
	}

	return id.Seal(f.Identity.X25519PublicKey, data)
}

type PushUser struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

type PushPayload struct {
	Config *config.Config    `json:"config"`
	Files  map[string]string `json:"files,omitempty"`
	Users  []PushUser        `json:"users,omitempty"`
}

type PushResult struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Error string `json:"error,omitempty"`
}

func pushImportApply(client *http.Client, f *config.Friend, configData []byte) error {
	sealed, err := sealForFriend(f, configData)
	if err != nil {
		return err
	}

	baseURL, token := f.URL, f.Token

	req, err := http.NewRequest(http.MethodPut, baseURL+"/api/config/import", bytes.NewReader(sealed))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", types.SealedContentType)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("PUT config/import: %w", err)
	}

	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("PUT config/import: HTTP %d", resp.StatusCode)
	}

	req2, err := http.NewRequest(http.MethodPost, baseURL+"/api/config/apply", nil)
	if err != nil {
		return fmt.Errorf("create apply request: %w", err)
	}

	req2.Header.Set("Authorization", "Bearer "+token)

	resp2, err := client.Do(req2)
	if err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	_, _ = io.Copy(io.Discard, resp2.Body)
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("apply: HTTP %d", resp2.StatusCode)
	}

	return nil
}
