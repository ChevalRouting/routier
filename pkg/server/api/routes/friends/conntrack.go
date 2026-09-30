package friends

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// ConfigureConntrack godoc
// @Summary  Configure conntrackd HA with a friend (applies on both sides)
// @Tags friends
// @Produce json
// @Param name path string true "friend name"
// @Param body body types.ConfigureConntrackRequest true "conntrack config"
// @Success 200 {object} types.Response[types.ConfigureConntrackResult]
// @Security BearerAuth
// @Router /api/friends/{name}/conntrack [post]
func ConfigureConntrack(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	var req types.ConfigureConntrackRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

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

	if cfg.HA == nil {
		cfg.HA = &cfgpkg.HA{}
	}

	cfg.HA.Conntrackd = &cfgpkg.Conntrackd{
		Interface:    req.LocalInterface,
		Address:      req.LocalAddress,
		PeerIPs:      []string{req.FriendAddress},
		Port:         req.Port,
		AllowInbound: req.AllowInbound,
	}

	friendConntrackd := &cfgpkg.Conntrackd{
		Interface:    req.FriendInterface,
		Address:      req.FriendAddress,
		PeerIPs:      []string{req.LocalAddress},
		Port:         req.Port,
		AllowInbound: req.AllowInbound,
	}

	payload := managers.PushPayload{Config: &cfgpkg.Config{Version: cfg.Version, HA: &cfgpkg.HA{Conntrackd: friendConntrackd}}}

	cfgpkg.ResolveInterfaces(cfg)
	persist := func() error { return cfgstore.WriteLive(app.ConfigPath, cfg) }
	opts := managers.ApplyOptions{Source: "web", ConfigPath: app.ConfigPath}
	if err := managers.CoordinatedApply(r.Context(), f, payload, persist, cfg, friendcache.InterpolationVars(), opts); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "coordinated conntrack apply failed"))
		return
	}

	types.OK(w, types.ConfigureConntrackResult{Status: "applied"})
}
