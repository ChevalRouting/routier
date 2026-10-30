package friends

import (
	"net/http"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/managers"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// @Summary  Derive an IP tunnel to a friend
// @Tags friends
// @Produce json
// @Param name path string true "friend name"
// @Param body body types.DeriveTunnelRequest true "mode/endpoints/subnet"
// @Success 200 {object} types.Response[types.DeriveTunnelResult]
// @Security BearerAuth
// @Router /api/friends/{name}/tunnel [post]
func DeriveTunnel(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	var req types.DeriveTunnelRequest
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

	localAddr, err := friendspkg.ResolveEndpointAddr(friendspkg.InterfaceAddresses(cfg), req.LocalEndpoint.Interface, req.LocalEndpoint.Address)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "local endpoint"))
		return
	}

	friendAddr := req.FriendEndpoint.Address
	if friendAddr == "" {
		client := friendspkg.NewClient(f.URL, f.Token, f.TLSSkipVerify)
		remote, err := client.Interfaces(r.Context())
		if err != nil {
			cached, ok := friendcache.CachedInterfaces(name)
			if !ok {
				types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to fetch friend interfaces"))
				return
			}

			remote = cached
		}

		friendAddr, err = friendspkg.ResolveEndpointAddr(remote, req.FriendEndpoint.Interface, "")
		if err != nil {
			types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "friend endpoint"))
			return
		}
	}

	res, err := friendspkg.DeriveTunnel(friendspkg.TunnelDeriveParams{
		FriendName:    f.Name,
		LocalHostname: cfg.Hostname,
		Mode:          req.Mode,
		Subnet:        req.Subnet,
		LocalAddr:     localAddr,
		FriendAddr:    friendAddr,
	})
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "derive tunnel"))
		return
	}

	if cfg.Interfaces == nil {
		cfg.Interfaces = map[string]*cfgpkg.Interface{}
	}

	cfg.Interfaces[res.InterfaceName] = res.Local.Interface()

	payload := managers.PushPayload{Config: &cfgpkg.Config{
		Version:    cfg.Version,
		Interfaces: map[string]*cfgpkg.Interface{res.InterfaceName: res.Counterpart.Interface()},
	}}

	cfgpkg.ResolveInterfaces(cfg)
	persist, err := cfgstore.MarshalLive(cfg)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "invalid config"))
		return
	}

	opts := managers.ApplyOptions{Source: "web", ConfigPath: app.ConfigPath, PersistConfig: persist}
	warning, err := managers.CoordinatedApply(r.Context(), f, payload, cfg, friendcache.InterpolationVars(), opts)
	if err != nil {
		types.Error(log.Logger, w, cfgstore.ApplyError(err, http.StatusBadGateway, "coordinated tunnel apply failed"))
		return
	}

	if warning != "" {
		log.Warn().Str("friend", f.Name).Msg(warning)
	}

	types.OK(w, types.DeriveTunnelResult{Interface: res.InterfaceName, Friend: f.Name, Pushed: true})
}
