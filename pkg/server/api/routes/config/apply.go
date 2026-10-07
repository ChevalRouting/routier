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
)

// @Summary  Apply the staged config
// @Tags config
// @Produce json
// @Param layer query string false "configuration layer" Enums(advanced, simple)
// @Success 200 {object} types.Response[types.ApplyResult]
// @Failure 500 {object} types.Response[types.ApplyResult]
// @Security BearerAuth
// @Router /api/config/apply [post]
func Apply(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())
	layer, appErr := requestLayer(r)
	if appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	owner, exists, err := cfgstore.StagingLayer(app.ConfigPath, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read staging owner"))
		return
	}

	if exists && owner != layer.Name() {
		types.Err(http.StatusConflict, "configuration changes belong to the "+owner+" layer").Write(w)
		return
	}

	staged, err := cfgstore.ReadLayer(app.ConfigPath, username, layer.Name())
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
		result := applyFailureResult(err)
		if result.Status == "validation_failed" {
			types.OK(w, result)
		} else {
			(&types.Response[types.ApplyResult]{HTTPStatusCode: http.StatusInternalServerError, ErrorText: err.Error(), Result: &result}).Write(w)
		}

		return
	}

	carryDDNSKey(cfg, staged)

	if err := cfgstore.PromoteConfig(app.ConfigPath, staged); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "applied but failed to persist config"))
		return
	}

	types.OK(w, types.ApplyResult{Status: "applied", SnapID: res.SnapID, Warning: res.Warning})
}

func carryDDNSKey(cfg, staged *cfgpkg.Config) {
	if cfg == nil || cfg.DHCP == nil || cfg.DHCP.DDNS == nil {
		return
	}

	if staged == nil || staged.DHCP == nil || staged.DHCP.DDNS == nil {
		return
	}

	staged.DHCP.DDNS.Key = cfg.DHCP.DDNS.Key
	staged.DHCP.DDNS.Algorithm = cfg.DHCP.DDNS.Algorithm
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
