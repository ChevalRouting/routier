package friends

import (
	"net/http"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// @Summary  Remove a friend
// @Tags friends
// @Produce json
// @Param name path string true "friend name"
// @Param confirm query bool false "also remove tagged sections"
// @Success 200 {object} types.Response[types.FriendDeleteResult]
// @Security BearerAuth
// @Router /api/friends/{name} [delete]
func Delete(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	if friendspkg.Index(cfg, name) < 0 {
		types.Err(http.StatusNotFound, "friend \""+name+"\" not found").Write(w)
		return
	}

	tagged := friendspkg.TaggedSections(cfg, name)
	confirm := r.URL.Query().Get("confirm") == "true"
	if !confirm && len(tagged) > 0 {
		types.OK(w, types.FriendDeleteResult{Friend: name, Sections: tagged, Deleted: false})
		return
	}

	friendspkg.RemoveTagged(cfg, name)
	if _, err := friendspkg.Remove(cfg, name); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusNotFound, err, "remove friend"))
		return
	}

	if err := cfgstore.WriteLive(app.ConfigPath, cfg); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "save config"))
		return
	}

	friendcache.Forget(name)
	types.OK(w, types.FriendDeleteResult{Friend: name, Sections: tagged, Deleted: true})
}
