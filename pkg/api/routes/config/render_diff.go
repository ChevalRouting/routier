package config

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/api/friendcache"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/diffutil"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
)

// RenderDiff godoc
// @Summary  Per-rendered-file diff for the pending apply
// @Tags config
// @Produce json
// @Success 200 {object} types.Response[[]types.FileDiff]
// @Security BearerAuth
// @Router /api/config/render-diff [get]
func RenderDiff(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())
	vars := friendcache.InterpolationVars()

	cfgstore.Mu.RLock()
	live, liveErr := cfgpkg.Load(app.ConfigPath)
	cfgstore.Mu.RUnlock()

	staged, err := cfgstore.Read(app.ConfigPath, username)
	if err == nil {
		staged, err = cfgstore.Resolve(staged)
	}

	if err != nil {
		types.Err(http.StatusBadRequest, "config interpolation: "+err.Error()).Write(w)
		return
	}

	before := renderedFiles(live, liveErr, vars)
	after := renderedFiles(staged, nil, vars)

	types.OK(w, diffutil.Files(before, after))
}

func renderedFiles(cfg *cfgpkg.Config, loadErr error, vars map[string]friends.Vars) map[string]string {
	if loadErr != nil || cfg == nil {
		return map[string]string{}
	}

	cfgpkg.ResolveInterfaces(cfg)
	outputs, err := render.All(cfg, render.WithFriends(vars))
	if err != nil {
		return map[string]string{}
	}

	m := make(map[string]string, len(outputs))
	for _, o := range outputs {
		m[o.Name] = o.Content
	}

	return m
}
