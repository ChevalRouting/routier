package switches

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	webdb "github.com/ChevalRouting/routier/pkg/db"
)

func openTestCredentials(t *testing.T) (*Credentials, *webdb.DB, string) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "web.db")
	db, err := webdb.InitDB(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = db.Close() })

	creds, err := OpenCredentials(db, dbPath)
	if err != nil {
		t.Fatal(err)
	}

	return creds, db, dbPath
}

func TestCredentialsAreSealedPerSwitch(t *testing.T) {
	creds, db, dbPath := openTestCredentials(t)
	ctx := context.Background()
	if err := creds.Set(ctx, "crs310", map[string]string{"username": "cibi", "password": "s3cret"}); err != nil {
		t.Fatal(err)
	}

	sealed, err := webdb.SwitchCredentials(ctx, db, "crs310")
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Contains(sealed, []byte("s3cret")) {
		t.Fatal("password stored in clear")
	}

	info, err := os.Stat(filepath.Join(filepath.Dir(dbPath), VaultKeyFilename))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("vault key missing or too open: %v %v", info, err)
	}

	got, err := creds.Get(ctx, "crs310")
	if err != nil || got["password"] != "s3cret" || got["username"] != "cibi" {
		t.Fatalf("unexpected credentials %v %v", got, err)
	}

	if err := webdb.SetSwitchCredentials(ctx, db, "crs309", sealed); err != nil {
		t.Fatal(err)
	}

	if _, err := creds.Get(ctx, "crs309"); err == nil {
		t.Fatal("credentials sealed for another switch were accepted")
	}

	names, err := creds.Names(ctx)
	if err != nil || !names["crs310"] || len(names) != 2 {
		t.Fatalf("unexpected names %v %v", names, err)
	}

	if err := creds.Delete(ctx, "crs310"); err != nil {
		t.Fatal(err)
	}

	if _, err := creds.Get(ctx, "crs310"); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("unexpected error after delete: %v", err)
	}

	if err := creds.Delete(ctx, "crs310"); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("unexpected error deleting twice: %v", err)
	}
}
