package ui

import (
	"encoding/json"
	"net/http"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Save a UI layout
// @Tags ui
// @Accept json
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/ui/layout [put]
func PutLayout(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		types.Err(http.StatusBadRequest, "invalid json").Write(w)
		return
	}

	if err := webdb.SetUILayout(requests.DurableContext(r), app.DB, "routing_topology", string(body)); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to save layout"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
