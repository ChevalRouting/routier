package v1

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
)

// @Summary Get session VPN configuration
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} types.Response[config.VPNConfig]
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/vpn [get]
func handleV1GetVPN(w http.ResponseWriter, r *http.Request) {
	sess := v1SessionFromCtx(r.Context())
	types.OK(w, sess.Config.VPN)
}

// @Summary Replace session VPN configuration
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param body body config.VPNConfig true "VPN configuration"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/vpn [put]
func handleV1PutVPN(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, vpn config.VPNConfig) { c.VPN = vpn })
}
