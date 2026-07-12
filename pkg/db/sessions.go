package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"gopkg.in/yaml.v3"
)

func ensureSessionsSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		id          TEXT    PRIMARY KEY,
		username    TEXT    NOT NULL,
		created_at  INTEGER NOT NULL,
		base_dir    TEXT    NOT NULL DEFAULT '',
		config_yaml TEXT    NOT NULL
	)`)
	return err
}

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

func InsertSession(db *sql.DB, id, username string, createdAt time.Time, baseDir string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	_, err = db.Exec(
		`INSERT INTO sessions (id, username, created_at, base_dir, config_yaml) VALUES (?, ?, ?, ?, ?)`,
		id, username, createdAt.Unix(), baseDir, string(data),
	)
	return err
}

func LoadSession(db *sql.DB, id string) (*Session, error) {
	var (
		username   string
		createdAt  int64
		baseDir    string
		configYAML string
	)
	err := db.QueryRow(
		`SELECT username, created_at, base_dir, config_yaml FROM sessions WHERE id = ?`, id,
	).Scan(&username, &createdAt, &baseDir, &configYAML)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var cfg config.Config
	if err := yaml.Unmarshal([]byte(configYAML), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal session config: %w", err)
	}

	return &Session{
		ID:        id,
		Username:  username,
		CreatedAt: time.Unix(createdAt, 0).UTC(),
		BaseDir:   baseDir,
		Config:    &cfg,
	}, nil
}

func UpdateSession(db *sql.DB, id string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	res, err := db.Exec(`UPDATE sessions SET config_yaml = ? WHERE id = ?`, string(data), id)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteSession(db *sql.DB, id string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func ListUserSessions(db *sql.DB, username string) ([]*SessionInfo, error) {
	rows, err := db.Query(
		`SELECT id, created_at FROM sessions WHERE username = ? ORDER BY created_at DESC`, username,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	out := []*SessionInfo{}
	for rows.Next() {
		var id string
		var ts int64

		if err := rows.Scan(&id, &ts); err != nil {
			return nil, err
		}

		out = append(out, &SessionInfo{ID: id, Username: username, CreatedAt: time.Unix(ts, 0).UTC()})
	}

	return out, rows.Err()
}
