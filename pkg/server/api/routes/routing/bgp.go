package routing

import (
	"net"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/daemon/vtysh"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Detailed `show bgp neighbors <address>` output from FRR
// @Tags routing
// @Produce json
// @Param address query string true "neighbor IP address"
// @Success 200 {object} types.Response[string]
// @Security BearerAuth
// @Router /api/routing/bgp/neighbor [get]
func BGPNeighbor(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("address")
	if net.ParseIP(addr) == nil {
		types.Err(http.StatusBadRequest, "address must be a valid IP").Write(w)
		return
	}

	out, err := vtysh.BGPNeighbor(addr)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to query BGP neighbor"))
		return
	}

	types.OK(w, string(out))
}
