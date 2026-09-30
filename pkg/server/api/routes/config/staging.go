package config

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// DiscardStaging godoc
// @Summary  Discard staged changes
// @Tags config
// @Produce json
// @Param layer query string false "configuration layer" Enums(advanced, simple)
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/config/staging [delete]
func DiscardStaging(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())
	layer, appErr := requestLayer(r)
	if appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	if err := cfgstore.DiscardLayer(app.ConfigPath, username, layer.Name()); err != nil {
		types.Error(log.Logger, w, cfgstore.StagingError(err, "failed to discard staging config"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
