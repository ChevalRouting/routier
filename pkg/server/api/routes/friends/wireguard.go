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

// @Summary  Derive a WireGuard tunnel to a friend
// @Tags friends
// @Produce json
// @Param name path string true "friend name"
// @Param body body types.DeriveWireguardRequest true "endpoints/subnet"
// @Success 200 {object} types.Response[types.DeriveWireguardResult]
// @Security BearerAuth
// @Router /api/friends/{name}/wireguard [post]
func DeriveWireguard(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	var req types.DeriveWireguardRequest
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

	client := friendspkg.NewClient(f.URL, f.Token, f.TLSSkipVerify)
	friendAddr := req.FriendEndpoint.Address
	if friendAddr == "" {
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

	res, err := friendspkg.DeriveWireguard(friendspkg.WGDeriveParams{
		FriendName:    f.Name,
		LocalHostname: cfg.Hostname,
		Subnet:        req.Subnet,
		LocalAddr:     localAddr,
		LocalPort:     req.LocalEndpoint.ListenPort,
		FriendAddr:    friendAddr,
		FriendPort:    req.FriendEndpoint.ListenPort,
	})
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "derive wireguard"))
		return
	}

	if cfg.Wireguard == nil {
		cfg.Wireguard = map[string]*cfgpkg.Wireguard{}
	}

	cfg.Wireguard[res.InterfaceName] = res.Local

	payload := managers.PushPayload{Config: &cfgpkg.Config{
		Version:   cfg.Version,
		Wireguard: map[string]*cfgpkg.Wireguard{res.InterfaceName: res.Counterpart},
	}}

	cfgpkg.ResolveInterfaces(cfg)
	persist := func() error { return cfgstore.WriteLive(app.ConfigPath, cfg) }
	opts := managers.ApplyOptions{Source: "web", ConfigPath: app.ConfigPath}
	if err := managers.CoordinatedApply(r.Context(), f, payload, persist, cfg, friendcache.InterpolationVars(), opts); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "coordinated wireguard apply failed"))
		return
	}

	types.OK(w, types.DeriveWireguardResult{Interface: res.InterfaceName, Friend: f.Name, Pushed: true})
}
