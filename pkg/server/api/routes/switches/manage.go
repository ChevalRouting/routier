package switches

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ChevalRouting/routier/pkg/render"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	swengine "github.com/ChevalRouting/routier/pkg/switches"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

func switchName(w http.ResponseWriter, r *http.Request) (string, bool) {
	name, err := requests.MustParam[string](r, "name")
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "invalid switch name"))
		return "", false
	}

	return name, true
}

// @Summary  Store the credentials of a switch, sealed in the routier database
// @Tags switches
// @Produce json
// @Param name path string true "switch name"
// @Param body body types.SwitchCredentialsRequest true "credential fields of the switch model"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/switches/{name}/credentials [put]
func SetCredentials(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name, ok := switchName(w, r)
	if !ok {
		return
	}

	var req types.SwitchCredentialsRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

	values := make(map[string]string, len(req.Values))
	for key, value := range req.Values {
		if value = strings.TrimSpace(value); value != "" {
			values[key] = value
		}
	}

	if values["username"] == "" || values["password"] == "" {
		types.Error(log.Logger, w, types.NewError(http.StatusUnprocessableEntity, "username and password are required"))
		return
	}

	if err := app.Switches.Credentials().Set(r.Context(), name, values); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to store credentials"))
		return
	}

	log.Info().Str("switch", name).Str("user", appctx.UsernameFromContext(r.Context())).Msg("switch credentials stored")
	types.OK(w, types.StatusResponse{Status: "ok"})
}

// @Summary  Forget the credentials of a switch
// @Tags switches
// @Produce json
// @Param name path string true "switch name"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/switches/{name}/credentials [delete]
func DeleteCredentials(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name, ok := switchName(w, r)
	if !ok {
		return
	}

	err := app.Switches.Credentials().Delete(r.Context(), name)
	if errors.Is(err, swengine.ErrNoCredentials) {
		types.Error(log.Logger, w, types.NewError(http.StatusNotFound, "no credentials stored for this switch"))
		return
	}

	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to delete credentials"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}

// @Summary  Device calls the staged config would make on a switch
// @Tags switches
// @Produce json
// @Param name path string true "switch name"
// @Success 200 {object} types.Response[types.SwitchPlanPreview]
// @Security BearerAuth
// @Router /api/switches/{name}/plan [get]
func Plan(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	name, ok := switchName(w, r)
	if !ok {
		return
	}

	staged, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context()))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read staged config"))
		return
	}

	desired, ok := staged.Switches[name]
	if !ok || desired == nil {
		types.Error(log.Logger, w, types.NewError(http.StatusNotFound, "switch not found in the staged config"))
		return
	}

	steps, err := app.Switches.Plan(r.Context(), name, desired, render.ReadSwitchArtifacts()[name])
	if err != nil {
		types.Error(log.Logger, w, engineError(err, "could not plan the switch changes"))
		return
	}

	preview := types.SwitchPlanPreview{Steps: make([]string, 0, len(steps))}
	for _, step := range steps {
		preview.Steps = append(preview.Steps, step.Describe())
	}

	types.OK(w, preview)
}
