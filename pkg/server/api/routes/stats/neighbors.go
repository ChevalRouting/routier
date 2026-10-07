package stats

import (
	"net/http"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
)

// @Summary  Latest stored neighbor stats
// @Tags stats
// @Produce json
// @Success 200 {object} types.Response[types.NeighborStatsResponse]
// @Security BearerAuth
// @Router /api/stats/neighbors [get]
func Neighbors(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	maxTS, neighbors := webdb.LatestNeighbors(r.Context(), app.DB)
	if neighbors == nil {
		neighbors = []types.NeighborStat{}
	}

	types.OK(w, types.NeighborStatsResponse{Neighbors: neighbors, TS: maxTS})
}
