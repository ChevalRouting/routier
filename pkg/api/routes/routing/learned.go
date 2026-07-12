package routing

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/ChevalRouting/routier/pkg/vtysh"
)

// Learned godoc
// @Summary  Learned OSPF/BGP routes
// @Tags routing
// @Produce json
// @Success 200 {object} types.Response[types.LearnedRoutesResponse]
// @Security BearerAuth
// @Router /api/routing/learned [get]
func Learned(w http.ResponseWriter, _ *http.Request) {
	types.OK(w, types.LearnedRoutesResponse{
		OSPF: readLearnedRoutes("ospf"),
		BGP:  readLearnedRoutes("bgp"),
	})
}

type learnedNexthop struct {
	IP    string `json:"ip"`
	Iface string `json:"interfaceName"`
}

type learnedRouteEntry struct {
	Protocol string           `json:"protocol"`
	Distance int              `json:"distance"`
	Metric   int              `json:"metric"`
	Nexthops []learnedNexthop `json:"nexthops"`
}

func readLearnedRoutes(proto string) []types.LearnedRoute {
	var routes []types.LearnedRoute

	for _, af := range []string{"ip", "ipv6"} {
		out, err := vtysh.RoutesJSON(af)
		if err != nil || len(out) == 0 {
			continue
		}

		var table map[string][]learnedRouteEntry
		if err := json.Unmarshal(out, &table); err != nil {
			continue
		}

		for prefix, entries := range table {
			for _, e := range entries {
				if !strings.EqualFold(e.Protocol, proto) {
					continue
				}

				r := types.LearnedRoute{
					Prefix:   prefix,
					Protocol: e.Protocol,
					Distance: e.Distance,
					Metric:   e.Metric,
				}
				if len(e.Nexthops) > 0 {
					r.NextHop = e.Nexthops[0].IP
					r.Iface = e.Nexthops[0].Iface
				}

				routes = append(routes, r)
			}
		}
	}

	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Prefix < routes[j].Prefix
	})
	return routes
}
