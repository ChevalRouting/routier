package db

import (
	"database/sql"
	"time"

	"github.com/ChevalRouting/routier/pkg/types"
)

func ensureAnnouncementsSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS announcements (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		message     TEXT    NOT NULL,
		level       TEXT    NOT NULL DEFAULT 'info',
		enabled     INTEGER NOT NULL DEFAULT 1,
		dismissible INTEGER NOT NULL DEFAULT 1,
		created_at  INTEGER NOT NULL,
		updated_at  INTEGER NOT NULL
	)`)
	return err
}

func scanAnnouncements(rows *sql.Rows) ([]types.Announcement, error) {
	defer rows.Close()

	out := []types.Announcement{}
	for rows.Next() {
		var a types.Announcement
		if err := rows.Scan(&a.ID, &a.Message, &a.Level, &a.Enabled, &a.Dismissible, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}

		out = append(out, a)
	}

	return out, rows.Err()
}

func ListAnnouncements(db *sql.DB) ([]types.Announcement, error) {
	rows, err := db.Query("SELECT id, message, level, enabled, dismissible, created_at, updated_at FROM announcements ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return scanAnnouncements(rows)
}

func EnabledAnnouncements(db *sql.DB) ([]types.Announcement, error) {
	rows, err := db.Query("SELECT id, message, level, enabled, dismissible, created_at, updated_at FROM announcements WHERE enabled = 1 ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return scanAnnouncements(rows)
}

func CreateAnnouncement(db *sql.DB, a *types.Announcement) (int64, error) {
	now := time.Now().Unix()

	res, err := db.Exec(
		"INSERT INTO announcements (message, level, enabled, dismissible, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		a.Message, a.Level, a.Enabled, a.Dismissible, now, now,
	)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func UpdateAnnouncement(db *sql.DB, a *types.Announcement) error {
	_, err := db.Exec(
		"UPDATE announcements SET message = ?, level = ?, enabled = ?, dismissible = ?, updated_at = ? WHERE id = ?",
		a.Message, a.Level, a.Enabled, a.Dismissible, time.Now().Unix(), a.ID,
	)
	return err
}

func DeleteAnnouncement(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM announcements WHERE id = ?", id)
	return err
}
