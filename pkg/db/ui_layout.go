package db

import "database/sql"

func ensureUILayoutSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ui_layout (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	return err
}

func UILayout(db *sql.DB, key string) (string, error) {
	var val string
	err := db.QueryRow("SELECT value FROM ui_layout WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}

	return val, err
}

func SetUILayout(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		"INSERT INTO ui_layout (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}
