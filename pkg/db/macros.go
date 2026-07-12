package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Macro struct {
	ID          string
	Name        string
	Description string
	BaseYAML    string
	ModYAML     string
	Sections    []string
	CreatedAt   time.Time
	CreatedBy   string
	ApplyCount  int
	AppliedAt   *time.Time
}

func ensureMacrosSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS macros (
		id          TEXT    PRIMARY KEY,
		name        TEXT    NOT NULL UNIQUE,
		description TEXT    NOT NULL DEFAULT '',
		base_yaml   TEXT    NOT NULL,
		mod_yaml    TEXT    NOT NULL,
		sections    TEXT    NOT NULL DEFAULT '[]',
		created_at  INTEGER NOT NULL,
		created_by  TEXT    NOT NULL DEFAULT '',
		apply_count INTEGER NOT NULL DEFAULT 0,
		applied_at  INTEGER
	)`)
	return err
}

func NewMacroID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

const macroCols = `id, name, description, base_yaml, mod_yaml, sections, created_at, created_by, apply_count, applied_at`

func InsertMacro(db *sql.DB, m *Macro) error {
	sects, _ := json.Marshal(m.Sections)
	_, err := db.Exec(
		`INSERT INTO macros (id, name, description, base_yaml, mod_yaml, sections, created_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Name, m.Description, m.BaseYAML, m.ModYAML, string(sects),
		m.CreatedAt.Unix(), m.CreatedBy,
	)
	return err
}

func scanMacro(m *Macro, scan func(...any) error) error {
	var (
		ts        int64
		sectJSON  string
		appliedAt *int64
	)

	if err := scan(&m.ID, &m.Name, &m.Description, &m.BaseYAML, &m.ModYAML,
		&sectJSON, &ts, &m.CreatedBy, &m.ApplyCount, &appliedAt); err != nil {
		return err
	}

	m.CreatedAt = time.Unix(ts, 0).UTC()
	if appliedAt != nil {
		t := time.Unix(*appliedAt, 0).UTC()
		m.AppliedAt = &t
	}

	_ = json.Unmarshal([]byte(sectJSON), &m.Sections)
	return nil
}

func loadMacro(db *sql.DB, where string, arg any) (*Macro, error) {
	var m Macro
	err := scanMacro(&m, db.QueryRow(`SELECT `+macroCols+` FROM macros WHERE `+where, arg).Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func LoadMacro(db *sql.DB, id string) (*Macro, error) {
	return loadMacro(db, "id = ?", id)
}

func LoadMacroByName(db *sql.DB, name string) (*Macro, error) {
	return loadMacro(db, "name = ?", name)
}

func ListMacros(db *sql.DB) ([]*Macro, error) {
	rows, err := db.Query(`SELECT ` + macroCols + ` FROM macros ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var out []*Macro
	for rows.Next() {
		var m Macro
		if err := scanMacro(&m, rows.Scan); err != nil {
			return nil, err
		}

		out = append(out, &m)
	}

	return out, rows.Err()
}

func DeleteMacro(db *sql.DB, id string) error {
	res, err := db.Exec(`DELETE FROM macros WHERE id = ?`, id)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func MarkMacroApplied(db *sql.DB, id string) error {
	_, err := db.Exec(
		`UPDATE macros SET apply_count = apply_count + 1, applied_at = ? WHERE id = ?`,
		time.Now().Unix(), id,
	)
	return err
}
