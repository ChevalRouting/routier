package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ChevalRouting/routier/pkg/db/generated"
)

var ErrSwitchCredentialsNotFound = errors.New("switch credentials not found")

func SetSwitchCredentials(ctx context.Context, db *DB, name string, sealed []byte) error {
	return db.queries.UpsertSwitchCredentials(ctx, generated.UpsertSwitchCredentialsParams{Name: name, Sealed: sealed, UpdatedAt: time.Now().Unix()})
}

func SwitchCredentials(ctx context.Context, db *DB, name string) ([]byte, error) {
	sealed, err := db.queries.GetSwitchCredentials(ctx, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSwitchCredentialsNotFound
	}

	return sealed, err
}

func SwitchCredentialNames(ctx context.Context, db *DB) ([]string, error) {
	return db.queries.ListSwitchCredentialNames(ctx)
}

func DeleteSwitchCredentials(ctx context.Context, db *DB, name string) error {
	rows, err := db.queries.DeleteSwitchCredentials(ctx, name)
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrSwitchCredentialsNotFound
	}

	return nil
}
