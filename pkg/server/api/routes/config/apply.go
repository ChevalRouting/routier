package config

import (
	"errors"
	"net/http"
	"strings"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/state/failures"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// @Summary  Apply the staged config
// @Tags config
// @Produce json
// @Success 200 {object} types.Response[types.ApplyResult]
// @Failure 500 {object} types.Response[types.ApplyResult]
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

	persist, err := yaml.Marshal(staged)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "marshal config"))
		return
	}

	res, err := managers.Apply(r.Context(), cfg, friendcache.InterpolationVars(),
		managers.ApplyOptions{Source: "web", ConfigPath: app.ConfigPath, PersistConfig: persist}, managers.WatchdogTimeout)
	if errors.Is(err, managers.ErrApplyPending) {
		types.Error(log.Logger, w, cfgstore.ApplyError(err, http.StatusConflict, ""))
		return
	}

	if err != nil {
		result := applyFailureResult(err)
		if result.Status == "validation_failed" {
			types.OK(w, result)
		} else {
			(&types.Response[types.ApplyResult]{HTTPStatusCode: http.StatusInternalServerError, ErrorText: err.Error(), Result: &result}).Write(w)
		}

		return
	}

	types.OK(w, types.ApplyResult{Status: "applied", SnapID: res.SnapID, Warning: res.Warning})
}

func applyFailureResult(err error) types.ApplyResult {
	result := types.ApplyResult{Status: "apply_failed", Errors: []types.ArtifactError{{Message: err.Error()}}}
	var diagnostic *failures.ApplyError
	if errors.As(err, &diagnostic) {
		result.BundleID = diagnostic.BundleID
		result.LogID = diagnostic.LogID
	}

	var ve *failures.ValidationError
	if errors.As(err, &ve) {
		result.Status = "validation_failed"
		result.Errors = ve.Errors
		result.BundleID = ve.BundleID
	}

	return result
}
