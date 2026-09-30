package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/db/generated"
	"gopkg.in/yaml.v3"
)

type Session struct {
	ID        string
	Username  string
	CreatedAt time.Time
	BaseDir   string
	Config    *config.Config
}

type SessionInfo struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func InsertSession(ctx context.Context, db *DB, id, username string, createdAt time.Time, baseDir string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return db.queries.InsertSession(ctx, generated.InsertSessionParams{
		ID: id, Username: username, CreatedAt: createdAt.Unix(), BaseDir: baseDir, ConfigYaml: string(data),
	})
}

func LoadSession(ctx context.Context, db *DB, id string) (*Session, error) {
	row, err := db.queries.LoadSession(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var cfg config.Config
	if err := yaml.Unmarshal([]byte(row.ConfigYaml), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal session config: %w", err)
	}

	return &Session{
		ID:        row.ID,
		Username:  row.Username,
		CreatedAt: time.Unix(row.CreatedAt, 0).UTC(),
		BaseDir:   row.BaseDir,
		Config:    &cfg,
	}, nil
}

func UpdateSession(ctx context.Context, db *DB, id string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	n, err := db.queries.UpdateSession(ctx, generated.UpdateSessionParams{ConfigYaml: string(data), ID: id})
	if err != nil {
		return err
	}

	if n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteSession(ctx context.Context, db *DB, id string) error {
	return db.queries.DeleteSession(ctx, id)
}

func ListUserSessions(ctx context.Context, db *DB, username string) ([]*SessionInfo, error) {
	rows, err := db.queries.ListUserSessions(ctx, username)
	if err != nil {
		return nil, err
	}

	out := make([]*SessionInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, &SessionInfo{ID: row.ID, Username: row.Username, CreatedAt: time.Unix(row.CreatedAt, 0).UTC()})
	}

	return out, nil
}
