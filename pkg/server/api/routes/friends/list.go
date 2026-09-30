package friends

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// List godoc
// @Summary  List configured friends
// @Tags     friends
// @Produce  json
// @Success  200 {object} types.Response[[]types.FriendInfo]
// @Security BearerAuth
// @Router   /api/friends [get]
func List(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	out := make([]types.FriendInfo, 0, len(cfg.Friends))
	for _, f := range cfg.Friends {
		out = append(out, friendInfo(f))
	}

	types.OK(w, out)
}

// Status godoc
// @Summary  Friend liveness/status snapshot
// @Tags     friends
// @Produce  json
// @Success  200 {object} types.Response[[]types.FriendStatus]
// @Security BearerAuth
// @Router   /api/friends/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	snap := friendcache.StatusSnapshot()
	out := make([]types.FriendStatus, 0, len(cfg.Friends))
	for _, f := range cfg.Friends {
		if st, ok := snap[f.Name]; ok {
			out = append(out, st)
		} else {
			out = append(out, types.FriendStatus{Name: f.Name})
		}
	}

	types.OK(w, out)
}
