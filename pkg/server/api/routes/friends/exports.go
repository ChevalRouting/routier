package friends

import (
	"net/http"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  nft variables this node publishes to its friends
// @Tags friends
// @Produce json
// @Success 200 {object} types.Response[[]types.FriendVar]
// @Security BearerAuth
// @Router /api/friends/exports [get]
func Exports(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	cfgpkg.ResolveInterfaces(cfg)
	vars := render.NftVars(cfg)
	out := make([]types.FriendVar, 0, len(vars))
	for _, v := range vars {
		out = append(out, types.FriendVar{Name: v.Name, Value: v.Value})
	}

	types.OK(w, out)
}
