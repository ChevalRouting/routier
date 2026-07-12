package persistencetest

import (
	"path/filepath"
	"testing"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/tests/harness"
)

func TestSessionCRUD(t *testing.T) {
	db, err := webdb.InitDB(filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	defer db.Close()

	cfg := harness.LoadCfg(t, harness.MinimalConfig)

	if err := webdb.InsertSession(db, "sess-1", "admin", time.Now(), cfg.BaseDir, cfg); err != nil {
		t.Fatalf("insert: %v", err)
	}

	loaded, err := webdb.LoadSession(db, "sess-1")
	if err != nil || loaded == nil {
		t.Fatalf("load: %v (%v)", loaded, err)
	}

	if loaded.Username != "admin" || loaded.Config.Hostname != "rtr1" {
		t.Fatalf("unexpected session: %+v", loaded)
	}

	cfg.Hostname = "rtr2"
	if err := webdb.UpdateSession(db, "sess-1", cfg); err != nil {
		t.Fatalf("update: %v", err)
	}

	if again, _ := webdb.LoadSession(db, "sess-1"); again.Config.Hostname != "rtr2" {
		t.Fatalf("update not persisted: %+v", again)
	}

	list, err := webdb.ListUserSessions(db, "admin")
	if err != nil || len(list) != 1 || list[0].ID != "sess-1" {
		t.Fatalf("list = %+v, err %v", list, err)
	}

	if err := webdb.DeleteSession(db, "sess-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if gone, _ := webdb.LoadSession(db, "sess-1"); gone != nil {
		t.Fatalf("session not deleted: %+v", gone)
	}
}
