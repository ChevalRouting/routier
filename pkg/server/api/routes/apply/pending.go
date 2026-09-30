package apply

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Pending godoc
// @Summary  Pending apply awaiting confirmation
// @Tags apply
// @Produce json
// @Success 200 {object} types.Response[types.ApplyPendingResponse]
// @Security BearerAuth
// @Router /api/apply/pending [get]
func Pending(w http.ResponseWriter, _ *http.Request) {
	p, err := managers.PendingStatus()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read pending apply"))
		return
	}

	if p == nil {
		types.OK(w, types.ApplyPendingResponse{Pending: false})
		return
	}

	types.OK(w, types.ApplyPendingResponse{
		Pending:   true,
		SnapID:    p.SnapID,
		Timeout:   p.Timeout,
		Remaining: p.Remaining,
	})
}
