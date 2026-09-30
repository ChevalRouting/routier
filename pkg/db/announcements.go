package db

import (
	"context"
	"errors"
	"time"

	"github.com/ChevalRouting/routier/pkg/db/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

var ErrAnnouncementNotFound = errors.New("announcement not found")

func announcements(rows []generated.Announcement) []types.Announcement {
	out := make([]types.Announcement, 0, len(rows))
	for _, row := range rows {
		out = append(out, types.Announcement{
			ID: row.ID, Message: row.Message, Level: row.Level,
			Enabled: row.Enabled != 0, Dismissible: row.Dismissible != 0,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}

	return out
}

func ListAnnouncements(ctx context.Context, db *DB) ([]types.Announcement, error) {
	rows, err := db.queries.ListAnnouncements(ctx)
	if err != nil {
		return nil, err
	}

	return announcements(rows), nil
}

func EnabledAnnouncements(ctx context.Context, db *DB) ([]types.Announcement, error) {
	rows, err := db.queries.ListEnabledAnnouncements(ctx)
	if err != nil {
		return nil, err
	}

	return announcements(rows), nil
}

func CreateAnnouncement(ctx context.Context, db *DB, a *types.Announcement) (int64, error) {
	now := time.Now().Unix()

	res, err := db.queries.CreateAnnouncement(ctx, generated.CreateAnnouncementParams{
		Message: a.Message, Level: a.Level, Enabled: boolInt(a.Enabled), Dismissible: boolInt(a.Dismissible),
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func UpdateAnnouncement(ctx context.Context, db *DB, a *types.Announcement) error {
	rows, err := db.queries.UpdateAnnouncement(ctx, generated.UpdateAnnouncementParams{
		Message: a.Message, Level: a.Level, Enabled: boolInt(a.Enabled), Dismissible: boolInt(a.Dismissible),
		UpdatedAt: time.Now().Unix(), ID: a.ID,
	})
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrAnnouncementNotFound
	}

	return nil
}

func DeleteAnnouncement(ctx context.Context, db *DB, id int64) error {
	rows, err := db.queries.DeleteAnnouncement(ctx, id)
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrAnnouncementNotFound
	}

	return nil
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}

	return 0
}
