package switches

import (
	"net/http"
	"sort"

	"github.com/ChevalRouting/routier/pkg/config"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	"github.com/ChevalRouting/routier/pkg/server/api/switchcache"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Switch models known to this routier
// @Tags switches
// @Produce json
// @Success 200 {object} types.Response[[]types.SwitchModel]
// @Security BearerAuth
// @Router /api/switch-models [get]
func Models(w http.ResponseWriter, r *http.Request) {
	types.OK(w, appctx.FromContext(r.Context()).Switches.Catalog().All())
}

// @Summary  Applied switches with their live state
// @Tags switches
// @Produce json
// @Success 200 {object} types.Response[[]types.SwitchStatus]
// @Security BearerAuth
// @Router /api/switches [get]
func List(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	names, err := app.Switches.Credentials().Names(r.Context())
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read switch credentials"))
		return
	}

	out := make([]types.SwitchStatus, 0, len(cfg.Switches))
	for name, sw := range cfg.Switches {
		out = append(out, status(r.Context(), app, name, sw, names))
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	types.OK(w, out)
}

func appliedSwitch(w http.ResponseWriter, r *http.Request) (*appctx.App, string, *config.Switch, bool) {
	app := appctx.FromContext(r.Context())
	name, err := requests.MustParam[string](r, "name")
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "invalid switch name"))
		return nil, "", nil, false
	}

	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return nil, "", nil, false
	}

	sw, ok := cfg.Switches[name]
	if !ok || sw == nil {
		types.Error(log.Logger, w, types.NewError(http.StatusNotFound, "switch not found in the applied config"))
		return nil, "", nil, false
	}

	return app, name, sw, true
}

// @Summary  One applied switch with its live state
// @Tags switches
// @Produce json
// @Param name path string true "switch name"
// @Success 200 {object} types.Response[types.SwitchStatus]
// @Security BearerAuth
// @Router /api/switches/{name} [get]
func Get(w http.ResponseWriter, r *http.Request) {
	app, name, sw, ok := appliedSwitch(w, r)
	if !ok {
		return
	}

	names, err := app.Switches.Credentials().Names(r.Context())
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read switch credentials"))
		return
	}

	types.OK(w, status(r.Context(), app, name, sw, names))
}

// @Summary  Read the switch now
// @Tags switches
// @Produce json
// @Param name path string true "switch name"
// @Success 200 {object} types.Response[types.SwitchStatus]
// @Security BearerAuth
// @Router /api/switches/{name}/refresh [post]
func Refresh(w http.ResponseWriter, r *http.Request) {
	app, name, sw, ok := appliedSwitch(w, r)
	if !ok {
		return
	}

	switchcache.Refresh(r.Context(), app.Switches, name, sw)
	names, err := app.Switches.Credentials().Names(r.Context())
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read switch credentials"))
		return
	}

	types.OK(w, status(r.Context(), app, name, sw, names))
}
