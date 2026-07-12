package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestInitDBEnablesIncrementalAutoVacuum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web.db")
	db, err := InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var mode int
	if err := db.QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		t.Fatal(err)
	}

	if mode != 2 {
		t.Fatalf("auto_vacuum = %d, want 2 (INCREMENTAL)", mode)
	}
}

func TestInitDBConvertsExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web.db")

	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := legacy.Exec(`CREATE TABLE legacy (id INTEGER)`); err != nil {
		t.Fatal(err)
	}

	legacy.Close()

	db, err := InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var mode int
	if err := db.QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		t.Fatal(err)
	}

	if mode != 2 {
		t.Fatalf("auto_vacuum = %d, want 2 after conversion", mode)
	}
}
