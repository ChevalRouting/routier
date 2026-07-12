package workers

import (
	"database/sql"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
)

func StartCleanup(db *sql.DB) {
	go func() {
		webdb.PruneOldStats(db)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			webdb.PruneOldStats(db)
		}
	}()
}
