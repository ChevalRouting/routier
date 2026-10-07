package snapshots

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// @Summary  Restore a snapshot
// @Tags snapshots
// @Produce json
// @Param id path string true "snapshot id"
// @Success 200 {object} types.Response[types.ApplyResult]
// @Security BearerAuth
// @Router /api/snapshots/{id}/restore [post]
func Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resolved, err := managers.RollbackSource(id, "restore")
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "restore failed"))
		return
	}

	types.OK(w, types.ApplyResult{Status: "restored", SnapID: resolved})
}
