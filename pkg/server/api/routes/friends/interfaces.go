package friends

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Interfaces godoc
// @Summary  This node's interface addresses
// @Tags friends
// @Produce json
// @Success 200 {object} types.Response[[]types.FriendInterface]
// @Security BearerAuth
// @Router /api/friends/interfaces [get]
func Interfaces(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	types.OK(w, friendspkg.InterfaceAddresses(cfg))
}

// RemoteInterfaces godoc
// @Summary  A friend's interface addresses (cached fallback)
// @Tags friends
// @Produce json
// @Param name path string true "friend name"
// @Success 200 {object} types.Response[[]types.FriendInterface]
// @Security BearerAuth
// @Router /api/friends/{name}/interfaces [get]
func RemoteInterfaces(w http.ResponseWriter, r *http.Request) {
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
	ifaces, err := client.Interfaces(r.Context())
	if ifaces == nil {
		ifaces = []types.FriendInterface{}
	}

	if err != nil {
		if cached, ok := friendcache.CachedInterfaces(name); ok {
			w.Header().Set("X-Friend-Stale", "true")
			types.OK(w, cached)
			return
		}

		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to fetch friend interfaces"))
		return
	}

	friendcache.CacheInterfaces(name, ifaces)
	types.OK(w, ifaces)
}
