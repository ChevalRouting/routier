package apply

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
)

// Confirm godoc
// @Summary  Confirm (keep) a pending apply
// @Tags apply
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/apply/confirm [post]
func Confirm(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	_, _ = managers.ConfirmPending()
	cfgstore.Discard(app.ConfigPath, username)

	types.OK(w, types.StatusResponse{Status: "confirmed"})
}
