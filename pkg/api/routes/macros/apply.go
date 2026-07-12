package macros

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/macro"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Apply godoc
// @Summary  Apply a macro to the live config
// @Tags macros
// @Produce json
// @Param id path string true "macro id"
// @Param body body types.MacroApplyRequest false "apply options"
// @Success 200 {object} types.Response[types.ApplyResult]
// @Security BearerAuth
// @Router /api/macros/{id}/apply [post]
func Apply(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req types.MacroApplyRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	m, err := webdb.LoadMacro(app.DB, id)
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

	if !req.Force {
		conflicts := macro.DetectConflicts(base, live, mod, m.Sections)
		if len(conflicts) > 0 {
			details := make([]types.ConflictDetail, len(conflicts))
			for i, c := range conflicts {
				details[i] = types.ConflictDetail{Section: c.Section, Key: c.Key, Message: c.Message}
			}

			(&types.Response[types.MacroConflictsResponse]{
				HTTPStatusCode: http.StatusConflict,
				AppCode:        http.StatusConflict,
				ErrorText:      "conflicts detected",
				Result:         &types.MacroConflictsResponse{Conflicts: details},
			}).Write(w)
			return
		}
	}

	merged := macro.ApplyDelta(base, live, mod, m.Sections)

	if errs := config.Validate(merged, true); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}

		types.Err(http.StatusBadRequest, "validation failed: "+strings.Join(msgs, "; ")).Write(w)
		return
	}

	cfgstore.Mu.RLock()
	oldData, _ := os.ReadFile(app.ConfigPath)
	cfgstore.Mu.RUnlock()

	newData, err := yaml.Marshal(merged)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "marshal merged config"))
		return
	}

	cfgstore.Mu.Lock()
	if err := os.MkdirAll(filepath.Dir(app.ConfigPath), 0755); err != nil {
		cfgstore.Mu.Unlock()
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "create config dir"))
		return
	}

	if err := os.WriteFile(app.ConfigPath, newData, 0600); err != nil {
		cfgstore.Mu.Unlock()
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "write config"))
		return
	}

	cfgstore.Mu.Unlock()

	res, err := managers.Apply(r.Context(), merged, friendcache.InterpolationVars(),
		managers.ApplyOptions{Source: "macro:" + m.Name, ConfigPath: app.ConfigPath}, managers.WatchdogTimeout)
	if err != nil {
		cfgstore.Mu.Lock()
		if len(oldData) > 0 {
			_ = os.WriteFile(app.ConfigPath, oldData, 0600)
		}

		cfgstore.Mu.Unlock()
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "apply failed (rolled back)"))
		return
	}

	_ = webdb.MarkMacroApplied(app.DB, id)

	types.OK(w, types.ApplyResult{Status: "applied", SnapID: res.SnapID, Warning: res.Warning})
}
