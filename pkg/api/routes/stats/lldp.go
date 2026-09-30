package stats

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
)

// LLDP godoc
// @Summary  Latest stored LLDP/CDP neighbors
// @Tags stats
// @Produce json
// @Success 200 {object} types.Response[types.LLDPStatsResponse]
// @Security BearerAuth
// @Router /api/stats/lldp [get]
func LLDP(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	maxTS, neighbors := webdb.LatestLLDP(r.Context(), app.DB)
	if neighbors == nil {
		neighbors = []types.LLDPNeighbor{}
	}

	types.OK(w, types.LLDPStatsResponse{Neighbors: neighbors, TS: maxTS})
}
