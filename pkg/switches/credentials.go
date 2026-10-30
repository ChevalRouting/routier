package switches

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ChevalRouting/routier/pkg/auth/vault"
	webdb "github.com/ChevalRouting/routier/pkg/db"
)

const VaultKeyFilename = "vault.key"

var ErrNoCredentials = errors.New("no credentials stored for this switch")

type Credentials struct {
	db    *webdb.DB
	vault *vault.Vault
}

func OpenCredentials(db *webdb.DB, dbPath string) (*Credentials, error) {
	keys, err := vault.LoadOrCreate(filepath.Join(filepath.Dir(dbPath), VaultKeyFilename))
	if err != nil {
		return nil, fmt.Errorf("switch credentials vault: %w", err)
	}

	return &Credentials{db: db, vault: keys}, nil
}

func aad(name string) []byte {
	return []byte("switch:" + name)
}

func (c *Credentials) Set(ctx context.Context, name string, values map[string]string) error {
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}

	sealed, err := c.vault.Seal(data, aad(name))
	if err != nil {
		return err
	}

	return webdb.SetSwitchCredentials(ctx, c.db, name, sealed)
}

func (c *Credentials) Get(ctx context.Context, name string) (map[string]string, error) {
	sealed, err := webdb.SwitchCredentials(ctx, c.db, name)
	if errors.Is(err, webdb.ErrSwitchCredentialsNotFound) {
		return nil, ErrNoCredentials
	}

	if err != nil {
		return nil, err
	}

	data, err := c.vault.Open(sealed, aad(name))
	if err != nil {
		return nil, fmt.Errorf("open credentials of switch %s: %w", name, err)
	}

	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}

	return values, nil
}

func (c *Credentials) Names(ctx context.Context) (map[string]bool, error) {
	names, err := webdb.SwitchCredentialNames(ctx, c.db)
	if err != nil {
		return nil, err
	}

	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}

	return set, nil
}

func (c *Credentials) Delete(ctx context.Context, name string) error {
	err := webdb.DeleteSwitchCredentials(ctx, c.db, name)
	if errors.Is(err, webdb.ErrSwitchCredentialsNotFound) {
		return ErrNoCredentials
	}

	return err
}
