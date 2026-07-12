package ha

import (
	"fmt"
	"net/http"
	"sort"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/conntrack"
	"github.com/ChevalRouting/routier/pkg/keepalived"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

type vrrpEntry struct {
	ifName string
	vrrp   *config.VRRPInstance
}

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

	var entries []vrrpEntry
	for ifName, iface := range cfg.Interfaces {
		for _, v := range iface.VRRP {
			entries = append(entries, vrrpEntry{ifName, v})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ifName != entries[j].ifName {
			return entries[i].ifName < entries[j].ifName
		}

		return entries[i].vrrp.ID < entries[j].vrrp.ID
	})

	for _, e := range entries {
		ifName := e.ifName
		if e.vrrp.Interface != "" {
			ifName = e.vrrp.Interface
		}

		instanceName := fmt.Sprintf("VI_%s_%d", e.ifName, e.vrrp.ID)
		if e.vrrp.Name != "" {
			instanceName = e.vrrp.Name
		}

		kd := states[instanceName]
		st := types.VRRPInstanceStatus{
			Name:      e.vrrp.Name,
			Interface: ifName,
			ID:        e.vrrp.ID,
			VIPs:      e.vrrp.VIPs,
			Priority:  e.vrrp.Priority,
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

	if cfg.Conntrackd != nil {
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
