package persistencetest

import (
	"path/filepath"
	"testing"

	webdb "github.com/ChevalRouting/routier/pkg/db"
)

func TestInitDBAdoptsExistingSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web.db")
	legacy, err := webdb.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := legacy.Exec(`CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}

	if _, err := legacy.Exec(`INSERT INTO settings (key, value) VALUES ('existing', 'preserved')`); err != nil {
		t.Fatal(err)
	}

	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	database, err := webdb.InitDB(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}

	if got := webdb.Setting(t.Context(), database, "existing", ""); got != "preserved" {
		t.Fatalf("setting = %q, want preserved", got)
	}

	var version int
	if err := database.QueryRow(`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1`).Scan(&version); err != nil {
		t.Fatal(err)
	}

	if version != 2 {
		t.Fatalf("migration version = %d, want 2", version)
	}

	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	database, err = webdb.InitDB(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}

	_ = database.Close()
}
