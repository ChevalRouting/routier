package ui

import (
	"net/http"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Get a saved UI layout
// @Tags ui
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/ui/layout [get]
func GetLayout(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	val, err := webdb.UILayout(r.Context(), app.DB, "routing_topology")
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read layout"))
		return
	}

	if val == "" {
		types.OK(w, map[string]any{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(val))
}
