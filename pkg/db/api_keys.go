package db

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

var ErrAPIKeyNotFound = errors.New("API key not found")

func apiKey(row generated.ListAPIKeysRow) types.APIKey {
	return types.APIKey{
		ID: row.ID, Name: row.Name, Prefix: row.Prefix,
		CreatedAt: row.CreatedAt, CreatedBy: row.CreatedBy, RevokedAt: row.RevokedAt,
	}
}

func CreateAPIKey(ctx context.Context, db *DB, key types.APIKey, token string) error {
	hash := sha256.Sum256([]byte(token))
	return db.queries.CreateAPIKey(ctx, generated.CreateAPIKeyParams{
		ID: key.ID, Name: key.Name, TokenHash: hash[:], Prefix: key.Prefix,
		CreatedAt: key.CreatedAt, CreatedBy: key.CreatedBy,
	})
}

func ListAPIKeys(ctx context.Context, db *DB) ([]types.APIKey, error) {
	rows, err := db.queries.ListAPIKeys(ctx)
	if err != nil {
		return nil, err
	}

	keys := make([]types.APIKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, apiKey(row))
	}

	return keys, nil
}

func AuthenticateAPIKey(ctx context.Context, db *DB, token string) (*types.APIKey, error) {
	hash := sha256.Sum256([]byte(token))
	row, err := db.queries.GetActiveAPIKeyByHash(ctx, hash[:])
	if err != nil {
		return nil, err
	}

	key := types.APIKey{
		ID: row.ID, Name: row.Name, Prefix: row.Prefix,
		CreatedAt: row.CreatedAt, CreatedBy: row.CreatedBy, RevokedAt: row.RevokedAt,
	}
	return &key, nil
}

func RevokeAPIKey(ctx context.Context, db *DB, id string) error {
	revokedAt := time.Now().Unix()
	rows, err := db.queries.RevokeAPIKey(ctx, generated.RevokeAPIKeyParams{RevokedAt: &revokedAt, ID: id})
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}
