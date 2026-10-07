package routing

import (
	"net/http"
	"strings"

	"github.com/ChevalRouting/routier/pkg/net/iproute"
	"github.com/ChevalRouting/routier/pkg/types"
)

// @Summary  Kernel neighbor (ARP/NDP) table
// @Tags routing
// @Produce json
// @Param q query string false "text filter"
// @Param family query string false "ip family"
// @Success 200 {object} types.Response[types.NeighborsResponse]
// @Security BearerAuth
// @Router /api/routing/neighbors [get]
func Neighbors(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	family := r.URL.Query().Get("family")

	raw := iproute.ShowNeighbors()
	neighbors := make([]types.Neighbor, 0, len(raw))
	for _, n := range raw {
		if family != "" && n.Family != family {
			continue
		}

		if q != "" {
			if !strings.Contains(strings.ToLower(n.Dst), q) &&
				!strings.Contains(strings.ToLower(n.Dev), q) &&
				!strings.Contains(strings.ToLower(n.LLAddr), q) &&
				!strings.Contains(strings.ToLower(n.State), q) {
				continue
			}
		}

		neighbors = append(neighbors, types.Neighbor{
			Dst:    n.Dst,
			Dev:    n.Dev,
			LLAddr: n.LLAddr,
			State:  n.State,
			Family: n.Family,
		})
	}

	types.OK(w, types.NeighborsResponse{Neighbors: neighbors})
}
