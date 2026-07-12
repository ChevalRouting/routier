package db

import "database/sql"

const (
	SettingPasswordChanged    = "password_changed"
	SettingOnboardingComplete = "onboarding_complete"
	SettingOnboardingState    = "onboarding_state"
)

func ensureSettingsSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	return err
}

func Setting(db *sql.DB, key, def string) string {
	var v string
	if err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v); err != nil {
		return def
	}

	return v
}

func SetSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}
