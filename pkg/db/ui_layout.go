package db

import (
	"context"
	"database/sql"

	"github.com/ChevalRouting/routier/pkg/db/generated"
)

func UILayout(ctx context.Context, db *DB, key string) (string, error) {
	val, err := db.queries.GetUILayout(ctx, key)
	if err == sql.ErrNoRows {
		return "", nil
	}

	return val, err
}

func SetUILayout(ctx context.Context, db *DB, key, value string) error {
	return db.queries.SetUILayout(ctx, generated.SetUILayoutParams{Key: key, Value: value})
}
