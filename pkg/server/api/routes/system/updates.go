package system

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/host/updates"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const updatesCacheTTL = 30 * time.Minute

// @Summary  List available package upgrades
// @Tags system
// @Produce json
// @Param refresh query bool false "force a fresh apk index refresh"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/system/updates [get]
func Updates(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	ctx := requests.DurableContext(r)

	if r.URL.Query().Get("refresh") != "1" {
		if cached, ok := cachedUpdates(ctx, app); ok {
			types.OK(w, cached)
			return
		}
	}

	avail, err := updates.Check(ctx)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to check for updates"))
		return
	}

	if data, mErr := json.Marshal(avail); mErr == nil {
		_ = webdb.SetSetting(ctx, app.DB, webdb.SettingSystemUpdates, string(data))
	}

	types.OK(w, avail)
}

func cachedUpdates(ctx context.Context, app *appctx.App) (updates.Available, bool) {
	raw := webdb.Setting(ctx, app.DB, webdb.SettingSystemUpdates, "")
	if raw == "" {
		return updates.Available{}, false
	}

	var avail updates.Available
	if err := json.Unmarshal([]byte(raw), &avail); err != nil {
		return updates.Available{}, false
	}

	if time.Since(avail.CheckedAt) > updatesCacheTTL {
		return updates.Available{}, false
	}

	return avail, true
}
