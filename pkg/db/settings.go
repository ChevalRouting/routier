package db

import (
	"context"
	"github.com/ChevalRouting/routier/pkg/db/generated"
)

const (
	SettingOnboardingComplete = "onboarding_complete"
	SettingOnboardingState    = "onboarding_state"
	SettingSystemUpdates      = "system_updates"
)

func Setting(ctx context.Context, db *DB, key, def string) string {
	v, err := db.queries.GetSetting(ctx, key)
	if err != nil {
		return def
	}

	return v
}

func SetSetting(ctx context.Context, db *DB, key, value string) error {
	return db.queries.SetSetting(ctx, generated.SetSettingParams{Key: key, Value: value})
}
