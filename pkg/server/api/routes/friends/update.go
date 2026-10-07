package friends

import (
	"net/http"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// @Summary  Update a friend
// @Tags friends
// @Produce json
// @Param name path string true "friend name"
// @Param body body types.UpdateFriendRequest true "fields"
// @Success 200 {object} types.Response[types.FriendInfo]
// @Security BearerAuth
// @Router /api/friends/{name} [put]
func Update(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name := chi.URLParam(r, "name")

	var req types.UpdateFriendRequest
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

	if req.URL != nil {
		f.URL = *req.URL
	}

	if req.Token != nil {
		f.Token = *req.Token
	}

	if req.TLSSkipVerify != nil {
		f.TLSSkipVerify = *req.TLSSkipVerify
	}

	if req.Enabled != nil {
		f.Enabled = req.Enabled
	}

	if req.Manage != nil {
		f.Manage = *req.Manage
	}

	if err := cfgstore.WriteLive(app.ConfigPath, cfg); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "save config"))
		return
	}

	types.OK(w, friendInfo(f))
}
