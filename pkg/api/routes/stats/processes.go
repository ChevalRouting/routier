package stats

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/api/workers"
	"github.com/ChevalRouting/routier/pkg/types"
)

// Processes godoc
// @Summary  Live process list
// @Tags stats
// @Produce json
// @Success 200 {object} types.Response[types.ProcessesResponse]
// @Security BearerAuth
// @Router /api/stats/processes [get]
func Processes(w http.ResponseWriter, _ *http.Request) {
	procs := workers.LiveSnapshot().Procs
	if procs == nil {
		procs = []types.ProcessInfo{}
	}

	types.OK(w, types.ProcessesResponse{Processes: procs})
}
