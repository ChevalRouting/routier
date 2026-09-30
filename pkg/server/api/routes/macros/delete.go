package macros

import (
	"database/sql"
	"errors"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Delete godoc
// @Summary  Delete a macro
// @Tags macros
// @Produce json
// @Param id path string true "macro id"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/macros/{id} [delete]
func Delete(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := webdb.DeleteMacro(requests.DurableContext(r), app.DB, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			types.Err(http.StatusNotFound, "macro not found").Write(w)
			return
		}

		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to delete macro"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "deleted"})
}
