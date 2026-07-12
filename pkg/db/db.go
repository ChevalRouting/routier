package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	ensureAutoVacuum(db)

	if err := ensureUsersSchema(db); err != nil {
		return nil, fmt.Errorf("create users table: %w", err)
	}

	if err := ensureUILayoutSchema(db); err != nil {
		return nil, fmt.Errorf("create ui_layout table: %w", err)
	}

	if err := ensureSettingsSchema(db); err != nil {
		return nil, fmt.Errorf("create settings table: %w", err)
	}

	if err := ensureKernelRoutesSchema(db); err != nil {
		return nil, fmt.Errorf("create kernel_routes table: %w", err)
	}

	if err := ensureIfaceStatsSchema(db); err != nil {
		return nil, fmt.Errorf("create iface_stats table: %w", err)
	}

	if err := ensureSystemStatsSchema(db); err != nil {
		return nil, fmt.Errorf("create system_stats table: %w", err)
	}

	if err := ensureBGPStatsSchema(db); err != nil {
		return nil, fmt.Errorf("create bgp_peer_stats table: %w", err)
	}

	if err := ensureProtoStatsSchema(db); err != nil {
		return nil, fmt.Errorf("create proto_stats table: %w", err)
	}

	if err := ensureNeighborStatsSchema(db); err != nil {
		return nil, fmt.Errorf("create neighbor_stats table: %w", err)
	}

	if err := ensureSessionsSchema(db); err != nil {
		return nil, fmt.Errorf("create sessions table: %w", err)
	}

	if err := ensureMacrosSchema(db); err != nil {
		return nil, fmt.Errorf("create macros table: %w", err)
	}

	if err := ensureAnnouncementsSchema(db); err != nil {
		return nil, fmt.Errorf("create announcements table: %w", err)
	}

	if err := seedDefaultUser(db); err != nil {
		return nil, err
	}

	return db, nil
}

func ensureAutoVacuum(db *sql.DB) {
	var mode int
	if err := db.QueryRow(`PRAGMA auto_vacuum`).Scan(&mode); err != nil {
		return
	}

	if mode == 2 {
		return
	}

	if _, err := db.Exec(`PRAGMA auto_vacuum = INCREMENTAL`); err != nil {
		return
	}

	_, _ = db.Exec(`VACUUM`)
}
