package persistencetest

import (
	"database/sql"
	"github.com/ChevalRouting/routier/tests/harness"
	"path/filepath"
	"testing"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/macro"
	_ "modernc.org/sqlite"
)

func macroDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := webdb.InitDB(filepath.Join(t.TempDir(), "macros.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}

	t.Cleanup(func() { db.Close() })

	return db
}

const macroBase = `version: v3.0.0
hostname: gw
dns:
  nameservers: ["1.1.1.1"]
`

const macroMod = `version: v3.0.0
hostname: gw
dns:
  nameservers: ["8.8.8.8"]
`

func TestMacroLifecycle(t *testing.T) {
	db := macroDB(t)

	base := harness.LoadCfg(t, macroBase)
	mod := harness.LoadCfg(t, macroMod)

	sections := macro.ComputeSections(base, mod)
	if len(sections) != 1 || sections[0] != "dns" {
		t.Fatalf("computed sections = %v, want [dns]", sections)
	}

	m := &webdb.Macro{
		ID:        webdb.NewMacroID(),
		Name:      "switch-dns",
		BaseYAML:  macroBase,
		ModYAML:   macroMod,
		Sections:  sections,
		CreatedAt: time.Now(),
		CreatedBy: "tester",
	}
	if err := webdb.InsertMacro(db, m); err != nil {
		t.Fatalf("insert: %v", err)
	}

	list, err := webdb.ListMacros(db)
	if err != nil || len(list) != 1 || list[0].Name != "switch-dns" {
		t.Fatalf("list = %+v, err %v", list, err)
	}

	loaded, err := webdb.LoadMacro(db, m.ID)
	if err != nil || loaded.Name != "switch-dns" {
		t.Fatalf("load = %+v, err %v", loaded, err)
	}

	result := macro.ApplyDelta(base, base, mod, sections)
	if result.DNS == nil || len(result.DNS.Nameservers) != 1 || result.DNS.Nameservers[0] != "8.8.8.8" {
		t.Fatalf("apply delta did not switch dns: %+v", result.DNS)
	}

	if err := webdb.MarkMacroApplied(db, m.ID); err != nil {
		t.Fatalf("mark applied: %v", err)
	}

	if again, _ := webdb.LoadMacro(db, m.ID); again.ApplyCount != 1 {
		t.Fatalf("apply count = %d, want 1", again.ApplyCount)
	}

	if err := webdb.DeleteMacro(db, m.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if list, _ := webdb.ListMacros(db); len(list) != 0 {
		t.Fatalf("macro not deleted: %+v", list)
	}
}
