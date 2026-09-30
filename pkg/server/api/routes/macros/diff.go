package macros

import (
	"fmt"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/diffutil"
	"github.com/ChevalRouting/routier/pkg/macro"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

func parseMacroConfigs(m *webdb.Macro) (base, mod *config.Config, err error) {
	base = &config.Config{}
	if err = yaml.Unmarshal([]byte(m.BaseYAML), base); err != nil {
		return nil, nil, fmt.Errorf("parse base config: %w", err)
	}

	mod = &config.Config{}
	if err = yaml.Unmarshal([]byte(m.ModYAML), mod); err != nil {
		return nil, nil, fmt.Errorf("parse mod config: %w", err)
	}

	return base, mod, nil
}

// Diff godoc
// @Summary  Diff a macro against the live config
// @Tags macros
// @Produce json
// @Param id path string true "macro id"
// @Success 200 {object} types.Response[[]types.DiffLine]
// @Security BearerAuth
// @Router /api/macros/{id}/diff [get]
func Diff(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	m, err := webdb.LoadMacro(r.Context(), app.DB, id)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to load macro"))
		return
	}

	if m == nil {
		types.Err(http.StatusNotFound, "macro not found").Write(w)
		return
	}

	cfgstore.Mu.RLock()
	live, err := config.Load(app.ConfigPath)
	cfgstore.Mu.RUnlock()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	base, mod, err := parseMacroConfigs(m)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, err.Error()))
		return
	}

	merged := macro.ApplyDelta(base, live, mod, m.Sections)

	liveYAML, _ := yaml.Marshal(live)
	mergedYAML, _ := yaml.Marshal(merged)

	lines := diffutil.Lines(string(liveYAML), string(mergedYAML))

	types.OK(w, lines)
}
