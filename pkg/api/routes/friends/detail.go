package friends

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/friendcache"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// RemoteConfig godoc
// @Summary  Fetch a friend's live config (falls back to cache when down)
// @Tags     friends
// @Produce  json
// @Param    name path string true "friend name"
// @Success  200 {object} types.Response[config.Config]
// @Security BearerAuth
// @Router   /api/friends/{name}/config [get]
func RemoteConfig(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	f := friendspkg.Get(cfg, name)
	if f == nil {
		types.Err(http.StatusNotFound, "friend \""+name+"\" not found").Write(w)
		return
	}

	client := friendspkg.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	client.SetSealOpener(app.Identity, f.Identity.PublicKey)
	remote, err := client.Config(r.Context())
	if err != nil {
		if cached, ok := friendcache.CachedConfig(name); ok {
			w.Header().Set("X-Friend-Stale", "true")
			types.OK(w, cached)
			return
		}

		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to fetch friend config"))
		return
	}

	friendcache.CacheConfig(name, remote)
	types.OK(w, remote)
}

// Cache godoc
// @Summary  Cached state we hold for a friend (status, interfaces, config)
// @Tags     friends
// @Produce  json
// @Param    name path string true "friend name"
// @Success  200 {object} types.Response[friendcache.CachedState]
// @Security BearerAuth
// @Router   /api/friends/{name}/cache [get]
func Cache(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	types.OK(w, friendcache.CacheState(name))
}

// NftablesImport godoc
// @Summary  nftables variables imported from a friend
// @Tags     friends
// @Produce  json
// @Param    name path string true "friend name"
// @Success  200 {object} types.Response[[]render.NftVar]
// @Security BearerAuth
// @Router   /api/friends/{name}/nftables [get]
func NftablesImport(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	types.OK(w, render.FriendNftVars(cfg, name, render.WithFriends(friendcache.InterpolationVars())))
}
