package ha

import (
	"fmt"
	"net/http"
	"sort"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/telemetry/conntrack"
	"github.com/ChevalRouting/routier/pkg/daemon/keepalived"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Status godoc
// @Summary  VRRP / conntrackd HA status
// @Tags ha
// @Produce json
// @Success 200 {object} types.Response[types.HAStatusResponse]
// @Security BearerAuth
// @Router /api/ha/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	resp := types.HAStatusResponse{
		VRRP: []types.VRRPInstanceStatus{},
	}

	states := keepalived.States()

	var instances []*config.VRRPInstance
	if cfg.HA != nil {
		instances = append(instances, cfg.HA.VRRP...)
	}

	sort.Slice(instances, func(i, j int) bool {
		if instances[i].Interface != instances[j].Interface {
			return instances[i].Interface < instances[j].Interface
		}

		return instances[i].ID < instances[j].ID
	})

	for _, v := range instances {
		instanceName := fmt.Sprintf("VI_%s_%d", v.Interface, v.ID)
		if v.Name != "" {
			instanceName = v.Name
		}

		kd := states[instanceName]
		st := types.VRRPInstanceStatus{
			Name:      v.Name,
			Interface: v.Interface,
			ID:        v.ID,
			VIPs:      v.VIPs,
			Priority:  v.Priority,
			State:     "UNKNOWN",
		}
		if kd != nil {
			st.State = kd.State
			st.MasterIP = kd.MasterIP
			for _, p := range kd.Peers {
				st.Peers = append(st.Peers, types.VRRPPeer{
					IP:       p.IP,
					Priority: p.Priority,
					LastSeen: p.LastSeen,
				})
			}
		}

		resp.VRRP = append(resp.VRRP, st)
	}

	if cfg.HA != nil && cfg.HA.Conntrackd != nil {
		resp.Conntrackd = conntrackdStatus()
	}

	types.OK(w, resp)
}

func conntrackdStatus() *types.ConntrackdStatus {
	st := &types.ConntrackdStatus{}
	st.Running = svc.ServiceRunning("conntrackd")
	if n, err := conntrack.Count(); err == nil {
		st.Entries = n
	}

	if bd, err := conntrack.List(); err == nil {
		st.ByProto = bd.ByProto
		st.TCPStates = bd.TCPStates
	}

	return st
}
