package routing

import (
	"net/http"
	"strings"

	"github.com/ChevalRouting/routier/pkg/lldp"
	"github.com/ChevalRouting/routier/pkg/types"
)

// LLDP godoc
// @Summary  LLDP/CDP link-layer neighbors
// @Tags routing
// @Produce json
// @Param q query string false "text filter"
// @Success 200 {object} types.Response[types.LLDPNeighborsResponse]
// @Security BearerAuth
// @Router /api/routing/lldp [get]
func LLDP(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))

	raw := lldp.ShowNeighbors()
	neighbors := make([]types.LLDPNeighbor, 0, len(raw))
	for _, n := range raw {
		if q != "" {
			if !strings.Contains(strings.ToLower(n.LocalIface), q) &&
				!strings.Contains(strings.ToLower(n.ChassisName), q) &&
				!strings.Contains(strings.ToLower(n.ChassisID), q) &&
				!strings.Contains(strings.ToLower(n.PortID), q) &&
				!strings.Contains(strings.ToLower(n.MgmtIP), q) {
				continue
			}
		}

		neighbors = append(neighbors, types.LLDPNeighbor{
			LocalIface:   n.LocalIface,
			Protocol:     n.Protocol,
			ChassisID:    n.ChassisID,
			ChassisName:  n.ChassisName,
			SysDescr:     n.SysDescr,
			MgmtIP:       n.MgmtIP,
			PortID:       n.PortID,
			PortDescr:    n.PortDescr,
			Capabilities: n.Capabilities,
			VLAN:         n.VLAN,
			Age:          n.Age,
		})
	}

	types.OK(w, types.LLDPNeighborsResponse{Neighbors: neighbors})
}
