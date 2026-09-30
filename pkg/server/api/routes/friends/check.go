package friends

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Check godoc
// @Summary  Force a health check + info refresh for one friend
// @Tags     friends
// @Produce  json
// @Param    name path string true "friend name"
// @Success  200 {object} types.Response[types.FriendStatus]
// @Security BearerAuth
// @Router   /api/friends/{name}/check [post]
func Check(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	st, err := friendcache.CheckOne(r.Context(), app.ConfigPath, name)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusNotFound, err, "friend check failed"))
		return
	}

	types.OK(w, st)
}
