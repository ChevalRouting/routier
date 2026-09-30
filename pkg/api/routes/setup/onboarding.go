package setup

import (
	"encoding/json"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/requests"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// GetOnboarding godoc
// @Summary  Get saved onboarding state
// @Tags setup
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/setup/onboarding [get]
func GetOnboarding(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	raw := webdb.Setting(r.Context(), app.DB, webdb.SettingOnboardingState, "")
	if raw == "" {
		types.OK(w, map[string]any{})
		return
	}

	var state map[string]any
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		types.OK(w, map[string]any{})
		return
	}

	types.OK(w, state)
}

// PutOnboarding godoc
// @Summary  Save onboarding state
// @Tags setup
// @Accept json
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/setup/onboarding [put]
func PutOnboarding(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	var state map[string]any
	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "invalid json"))
		return
	}

	data, err := json.Marshal(state)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "marshal state"))
		return
	}

	if err := webdb.SetSetting(requests.DurableContext(r), app.DB, webdb.SettingOnboardingState, string(data)); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "save state"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
