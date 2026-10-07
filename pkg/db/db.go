package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type DB struct {
	*sql.DB
	queries *generated.Queries
}

func DSN(path string) string {
	return path + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
}

func Open(path string) (*DB, error) {
	database, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		return nil, err
	}

	return New(database), nil
}

func New(database *sql.DB) *DB {
	return &DB{DB: database, queries: generated.New(database)}
}

func InitDB(ctx context.Context, path string) (*DB, error) {
	db, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	ensureAutoVacuum(ctx, db.DB)

	migrationFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db.DB, migrationFS)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return db, nil
}

func ensureAutoVacuum(ctx context.Context, db *sql.DB) {
	var mode int
	if err := db.QueryRowContext(ctx, `PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		return
	}

	if mode == 2 {
		return
	}

	if _, err := db.ExecContext(ctx, `PRAGMA auto_vacuum = INCREMENTAL`); err != nil {
		return
	}

	_, _ = db.ExecContext(ctx, `VACUUM`)
}
