package macros

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Get godoc
// @Summary  Get a macro
// @Tags macros
// @Produce json
// @Param id path string true "macro id"
// @Success 200 {object} types.Response[types.MacroInfo]
// @Security BearerAuth
// @Router /api/macros/{id} [get]
func Get(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	m, err := webdb.LoadMacro(r.Context(), app.DB, id)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to load macro"))
		return
	}

	if m == nil {
		types.Err(http.StatusNotFound, "macro not found").Write(w)
		return
	}

	types.OK(w, macroToInfo(m))
}
