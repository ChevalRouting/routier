package friends

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Sync godoc
// @Summary  Sync HA state to one friend
// @Tags friends
// @Produce json
// @Param name path string true "friend name"
// @Success 200 {object} types.Response[[]managers.PushResult]
// @Security BearerAuth
// @Router /api/friends/{name}/sync [post]
func Sync(w http.ResponseWriter, r *http.Request) {
	syncHA(w, r, chi.URLParam(r, "name"))
}

// SyncAll godoc
// @Summary  Sync HA state to all friends
// @Tags friends
// @Produce json
// @Success 200 {object} types.Response[[]managers.PushResult]
// @Security BearerAuth
// @Router /api/friends/sync [post]
func SyncAll(w http.ResponseWriter, r *http.Request) {
	syncHA(w, r, "")
}

func syncHA(w http.ResponseWriter, r *http.Request, name string) {
	app := appctx.FromContext(r.Context())

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	results, err := managers.SyncHA(r.Context(), cfg, name)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "ha sync"))
		return
	}

	types.OK(w, results)
}
