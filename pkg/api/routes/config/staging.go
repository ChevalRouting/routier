package config

import (
	"net/http"
	"os"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// DiscardStaging godoc
// @Summary  Discard staged changes
// @Tags config
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/config/staging [delete]
func DiscardStaging(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())
	if err := os.Remove(cfgstore.StagingPath(app.ConfigPath, username)); err != nil && !os.IsNotExist(err) {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to discard staging"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
