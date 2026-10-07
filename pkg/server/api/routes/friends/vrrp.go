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

func upsertVRRP(list []*cfgpkg.VRRPInstance, inst *cfgpkg.VRRPInstance) []*cfgpkg.VRRPInstance {
	out := make([]*cfgpkg.VRRPInstance, 0, len(list)+1)
	for _, v := range list {
		if v.ID != inst.ID {
			out = append(out, v)
		}
	}

	return append(out, inst)
}

// @Summary  Configure a VRRP instance shared with a friend (applies on both sides)
// @Tags friends
// @Produce json
// @Param name path string true "friend name"
// @Param body body types.ConfigureVRRPRequest true "vrrp config"
// @Success 200 {object} types.Response[types.ConfigureVRRPResult]
// @Security BearerAuth
// @Router /api/friends/{name}/vrrp [post]
func ConfigureVRRP(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	var req types.ConfigureVRRPRequest
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

	if cfg.Interfaces == nil || cfg.Interfaces[req.LocalInterface] == nil {
		types.Err(http.StatusBadRequest, "local interface \""+req.LocalInterface+"\" not found").Write(w)
		return
	}

	ourPriority := req.Priority
	if ourPriority == 0 {
		ourPriority = 150
	}

	friendPriority := ourPriority - 50
	if friendPriority < 1 {
		friendPriority = 100
	}

	if cfg.HA == nil {
		cfg.HA = &cfgpkg.HA{}
	}

	cfg.HA.VRRP = upsertVRRP(cfg.HA.VRRP, &cfgpkg.VRRPInstance{ID: req.VRID, Interface: req.LocalInterface, VIPs: req.VIPs, Priority: ourPriority, Friend: name})

	var friendVrrp []*cfgpkg.VRRPInstance
	if fcfg, ok := friendcache.CachedConfig(name); ok && fcfg.HA != nil {
		for _, v := range fcfg.HA.VRRP {
			if v.Interface == req.FriendInterface {
				friendVrrp = append(friendVrrp, v)
			}
		}
	}

	friendVrrp = upsertVRRP(friendVrrp, &cfgpkg.VRRPInstance{ID: req.VRID, Interface: req.FriendInterface, VIPs: req.VIPs, Priority: friendPriority, Friend: cfg.Hostname})

	payload := managers.PushPayload{Config: &cfgpkg.Config{
		Version: cfg.Version,
		HA:      &cfgpkg.HA{VRRP: friendVrrp},
	}}

	cfgpkg.ResolveInterfaces(cfg)

	persist := func() error { return cfgstore.WriteLive(app.ConfigPath, cfg) }
	opts := managers.ApplyOptions{Source: "web", ConfigPath: app.ConfigPath}

	if err := managers.CoordinatedApply(r.Context(), f, payload, persist, cfg, friendcache.InterpolationVars(), opts); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "coordinated vrrp apply failed"))
		return
	}

	types.OK(w, types.ConfigureVRRPResult{Status: "applied"})
}
