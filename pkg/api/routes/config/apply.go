package config

import (
	"net/http"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/api/friendcache"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Apply godoc
// @Summary  Apply the staged config
// @Tags config
// @Produce json
// @Success 200 {object} types.Response[types.ApplyResult]
// @Security BearerAuth
// @Router /api/config/apply [post]
func Apply(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	staged, err := cfgstore.Read(app.ConfigPath, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	cfg, err := cfgstore.Resolve(staged)
	if err != nil {
		types.Err(http.StatusBadRequest, "config interpolation: "+err.Error()).Write(w)
		return
	}

	if errs := cfgpkg.Validate(cfg, true); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}

		types.Err(http.StatusBadRequest, "validation failed: "+strings.Join(msgs, "; ")).Write(w)
		return
	}

	res, err := managers.Apply(r.Context(), cfg, friendcache.InterpolationVars(),
		managers.ApplyOptions{Source: "web", ConfigPath: app.ConfigPath}, managers.WatchdogTimeout)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "apply failed (rolled back)"))
		return
	}

	if err := cfgstore.PromoteConfig(app.ConfigPath, staged); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "applied but failed to persist config"))
		return
	}

	types.OK(w, types.ApplyResult{Status: "applied", SnapID: res.SnapID, Warning: res.Warning})
}
