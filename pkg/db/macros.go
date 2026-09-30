package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ChevalRouting/routier/pkg/db/generated"
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

func NewMacroID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func InsertMacro(ctx context.Context, db *DB, m *Macro) error {
	sects, _ := json.Marshal(m.Sections)
	return db.queries.InsertMacro(ctx, generated.InsertMacroParams{
		ID: m.ID, Name: m.Name, Description: m.Description, BaseYaml: m.BaseYAML, ModYaml: m.ModYAML,
		Sections: string(sects), CreatedAt: m.CreatedAt.Unix(), CreatedBy: m.CreatedBy,
	})
}

func macro(row generated.Macro) *Macro {
	m := &Macro{
		ID: row.ID, Name: row.Name, Description: row.Description, BaseYAML: row.BaseYaml,
		ModYAML: row.ModYaml, CreatedAt: time.Unix(row.CreatedAt, 0).UTC(), CreatedBy: row.CreatedBy,
		ApplyCount: int(row.ApplyCount),
	}
	if row.AppliedAt != nil {
		t := time.Unix(*row.AppliedAt, 0).UTC()
		m.AppliedAt = &t
	}

	_ = json.Unmarshal([]byte(row.Sections), &m.Sections)
	return m
}

func LoadMacro(ctx context.Context, db *DB, id string) (*Macro, error) {
	row, err := db.queries.LoadMacro(ctx, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return macro(row), nil
}

func LoadMacroByName(ctx context.Context, db *DB, name string) (*Macro, error) {
	row, err := db.queries.LoadMacroByName(ctx, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return macro(row), nil
}

func ListMacros(ctx context.Context, db *DB) ([]*Macro, error) {
	rows, err := db.queries.ListMacros(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*Macro, 0, len(rows))
	for _, row := range rows {
		out = append(out, macro(row))
	}

	return out, nil
}

func DeleteMacro(ctx context.Context, db *DB, id string) error {
	n, err := db.queries.DeleteMacro(ctx, id)
	if err != nil {
		return err
	}

	if n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func MarkMacroApplied(ctx context.Context, db *DB, id string) error {
	appliedAt := time.Now().Unix()
	return db.queries.MarkMacroApplied(ctx, generated.MarkMacroAppliedParams{
		AppliedAt: &appliedAt, ID: id,
	})
}
