package switches

import (
	"net/http"
	"slices"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/switches/model"
	"github.com/ChevalRouting/routier/pkg/telemetry/lldp"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const routierSysDescr = "Routier"

// @Summary  Switches heard over LLDP or CDP and not configured yet
// @Tags switches
// @Produce json
// @Success 200 {object} types.Response[[]types.SwitchCandidate]
// @Security BearerAuth
// @Router /api/switches/candidates [get]
func Candidates(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	_, neighbors := webdb.LatestLLDP(r.Context(), app.DB)
	if len(neighbors) == 0 {
		neighbors = liveNeighbors()
	}

	types.OK(w, candidates(neighbors, cfg, app.Switches.Catalog()))
}

func liveNeighbors() []types.LLDPNeighbor {
	var out []types.LLDPNeighbor
	for _, n := range lldp.ShowNeighbors() {
		out = append(out, types.LLDPNeighbor{
			LocalIface: n.LocalIface, Protocol: n.Protocol, ChassisID: n.ChassisID, ChassisName: n.ChassisName,
			SysDescr: n.SysDescr, MgmtIP: n.MgmtIP, PortID: n.PortID, PortDescr: n.PortDescr, Capabilities: n.Capabilities,
		})
	}

	return out
}

func configuredHosts(cfg *config.Config) map[string]bool {
	hosts := make(map[string]bool, len(cfg.Switches))
	for _, sw := range cfg.Switches {
		if sw != nil {
			hosts[sw.Host] = true
		}
	}

	return hosts
}

func candidates(neighbors []types.LLDPNeighbor, cfg *config.Config, catalog *model.Catalog) []types.SwitchCandidate {
	hosts := configuredHosts(cfg)
	out := []types.SwitchCandidate{}
	index := map[string]int{}
	for _, n := range neighbors {
		if n.SysDescr == routierSysDescr || n.ChassisID == "" || (n.MgmtIP != "" && hosts[n.MgmtIP]) {
			continue
		}

		position, seen := index[n.ChassisID]
		if !seen {
			position = len(out)
			index[n.ChassisID] = position
			out = append(out, types.SwitchCandidate{ChassisID: n.ChassisID, LocalIface: n.LocalIface, PortID: n.PortID, Protocols: []string{}})
		}

		mergeCandidate(&out[position], n, catalog)
	}

	return out
}

func mergeCandidate(c *types.SwitchCandidate, n types.LLDPNeighbor, catalog *model.Catalog) {
	c.Name = firstNonEmpty(c.Name, n.ChassisName)
	c.SysDescr = firstNonEmpty(c.SysDescr, n.SysDescr)
	c.MgmtIP = firstNonEmpty(c.MgmtIP, n.MgmtIP)
	c.Capabilities = firstNonEmpty(c.Capabilities, n.Capabilities)
	if !slices.Contains(c.Protocols, n.Protocol) {
		c.Protocols = append(c.Protocols, n.Protocol)
	}

	if found, ok := catalog.Match(c.SysDescr); ok && c.ModelID == "" {
		c.ModelID = found.ID
	}
}

func firstNonEmpty(current, candidate string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}

	return candidate
}
