package cibi

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"time"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/rs/zerolog/log"
)

const (
	PendingFilename = "cibi-join.json"
	firstAttempt    = 10 * time.Second
	retryEvery      = 30 * time.Second
	giveUpAfter     = 24 * time.Hour
)

func SavePending(path string, j Join) error {
	if err := j.Validate(); err != nil {
		return err
	}

	data, err := json.Marshal(j)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func LoadPending(path string) (Join, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Join{}, false, nil
	}

	if err != nil {
		return Join{}, false, err
	}

	var j Join
	return j, true, json.Unmarshal(data, &j)
}

func JoinPending(ctx context.Context, db *webdb.DB, self *identity.Identity, path string) {
	j, found, err := LoadPending(path)
	if err != nil {
		log.Warn().Err(err).Str("path", path).Msg("read the pending cibi join")
		return
	}

	if !found || self == nil {
		return
	}

	deadline := time.Now().Add(giveUpAfter)
	wait := firstAttempt
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}

		result, err := j.Run(ctx, db, self)
		if err == nil {
			log.Info().Str("name", result.Name).Msg(result.Summary())
			_ = os.Remove(path)
			return
		}

		if errors.Is(err, ErrRefused) || time.Now().After(deadline) {
			log.Error().Err(err).Msg("joining cibi stopped")
			_ = os.Remove(path)
			return
		}

		log.Warn().Err(err).Dur("retry", retryEvery).Msg("joining cibi failed")
		wait = retryEvery
	}
}
