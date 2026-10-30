package db

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
)

func TestInitDBConcurrentMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web.db")
	errs := make([]error, 8)
	var wg sync.WaitGroup
	for index := range errs {
		wg.Go(func() { errs[index] = initAndClose(path) })
	}

	wg.Wait()
	for index, err := range errs {
		if err != nil {
			t.Fatalf("opener %d: %v", index, err)
		}
	}
}

func initAndClose(path string) error {
	db, err := InitDB(context.Background(), path)
	if err != nil {
		return err
	}

	return db.Close()
}
