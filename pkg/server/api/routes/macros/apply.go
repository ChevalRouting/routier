package macros

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/macro"
	"github.com/ChevalRouting/routier/pkg/managers"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

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
	username := appctx.UsernameFromContext(r.Context())
	staged, err := cfgstore.HasStaging(app.ConfigPath, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "read staging"))
		return
	}

	if staged {
		types.Err(http.StatusConflict, "apply or discard the staged changes before running a macro").Write(w)
		return
	}

	var req types.MacroApplyRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

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

	newData, err := yaml.Marshal(merged)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "marshal merged config"))
		return
	}

	res, err := managers.Apply(r.Context(), merged, friendcache.InterpolationVars(),
		managers.ApplyOptions{Source: "macro:" + m.Name, ConfigPath: app.ConfigPath, PersistConfig: newData}, managers.WatchdogTimeout)
	if err != nil {
		types.Error(log.Logger, w, cfgstore.ApplyError(err, http.StatusInternalServerError, "apply failed (rolled back)"))
		return
	}

	_ = webdb.MarkMacroApplied(requests.DurableContext(r), app.DB, id)

	types.OK(w, types.ApplyResult{Status: "applied", SnapID: res.SnapID, Warning: res.Warning})
}
