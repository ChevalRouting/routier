package apply

import (
	"io"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/applylog"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
)

// Log godoc
// @Summary  Read one apply log
// @Tags apply
// @Produce plain
// @Param id path string true "apply log id"
// @Success 200 {string} string
// @Security BearerAuth
// @Router /api/apply/logs/{id} [get]
func Log(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	text, err := applylog.Read(id)
	if err != nil {
		types.Err(http.StatusNotFound, "apply log not found").Write(w)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, text)
}
