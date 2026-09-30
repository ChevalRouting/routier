package stats

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/api/workers"
	"github.com/ChevalRouting/routier/pkg/telemetry/stats"
	"github.com/ChevalRouting/routier/pkg/types"
)

// Get godoc
// @Summary  Live system / interface / routing stats
// @Tags stats
// @Produce json
// @Success 200 {object} types.Response[types.StatsResponse]
// @Security BearerAuth
// @Router /api/stats [get]
func Get(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	snap := workers.LiveSnapshot()

	resp := types.StatsResponse{
		Interfaces: stats.ReadIfaceStats(),
		System:     snap.Sys,
	}
	resp.BGP = snap.BGP
	resp.OSPF = snap.OSPF

	if resp.BGP != nil && len(resp.BGP.Peers) > 0 {
		if cfg, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context())); err == nil &&
			cfg.Routing != nil && cfg.Routing.BGP != nil {
			descByAddr := make(map[string]string, len(cfg.Routing.BGP.Neighbors))
			for _, n := range cfg.Routing.BGP.Neighbors {
				if n.Description != "" {
					descByAddr[n.Address] = n.Description
				}
			}

			for i := range resp.BGP.Peers {
				resp.BGP.Peers[i].Description = descByAddr[resp.BGP.Peers[i].Address]
			}
		}
	}

	types.OK(w, resp)
}
