package apply

import (
	"errors"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/managers"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Confirm (keep) a pending apply
// @Tags apply
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/apply/confirm [post]
func Confirm(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	if _, err := managers.ConfirmPending(); err != nil && !errors.Is(err, managers.ErrNoPending) {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "confirm failed"))
		return
	}

	if err := cfgstore.DiscardApplied(app.ConfigPath, username); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "discard applied draft failed"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "confirmed"})
}
