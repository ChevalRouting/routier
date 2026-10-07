package persistencetest

import (
	"path/filepath"
	"testing"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestAnnouncementsCRUD(t *testing.T) {
	ctx := t.Context()
	db, err := webdb.InitDB(ctx, filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	defer func(action func() error) { _ = action() }(db.Close)

	id, err := webdb.CreateAnnouncement(ctx, db, &types.Announcement{Message: "maintenance", Level: "warning", Enabled: true, Dismissible: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	disabledID, err := webdb.CreateAnnouncement(ctx, db, &types.Announcement{Message: "draft", Level: "info", Enabled: false, Dismissible: true})
	if err != nil {
		t.Fatalf("create disabled: %v", err)
	}

	all, err := webdb.ListAnnouncements(ctx, db)
	if err != nil || len(all) != 2 {
		t.Fatalf("list all: %v, %d", err, len(all))
	}

	active, err := webdb.EnabledAnnouncements(ctx, db)
	if err != nil || len(active) != 1 || active[0].ID != id {
		t.Fatalf("expected only the enabled one active, got %+v (%v)", active, err)
	}

	if err := webdb.UpdateAnnouncement(ctx, db, &types.Announcement{ID: disabledID, Message: "now live", Level: "danger", Enabled: true, Dismissible: false}); err != nil {
		t.Fatalf("update: %v", err)
	}

	if active, _ := webdb.EnabledAnnouncements(ctx, db); len(active) != 2 {
		t.Fatalf("expected 2 active after enabling, got %d", len(active))
	}

	if err := webdb.DeleteAnnouncement(ctx, db, id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if all, _ := webdb.ListAnnouncements(ctx, db); len(all) != 1 {
		t.Fatalf("expected 1 after delete, got %d", len(all))
	}
}
