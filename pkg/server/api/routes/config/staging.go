package config

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/types"
)

// @Summary  Discard staged changes
// @Tags config
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/config/staging [delete]
func DiscardStaging(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())
	cfgstore.Discard(app.ConfigPath, username)

	types.OK(w, types.StatusResponse{Status: "ok"})
}
