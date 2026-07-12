package v1

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

type v1HostnameBody struct {
	Hostname string `json:"hostname"`
}

// handleV1GetHostname godoc
// @Summary  Get session hostname
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/hostname [get]
func handleV1GetHostname(w http.ResponseWriter, r *http.Request) {
	sess := v1SessionFromCtx(r.Context())
	types.OK(w, v1HostnameBody{Hostname: sess.Config.Hostname})
}

// handleV1PutHostname godoc
// @Summary  Set session hostname
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/hostname [put]
func handleV1PutHostname(w http.ResponseWriter, r *http.Request) {
	body, err := v1Decode[v1HostnameBody](r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if body.Hostname == "" {
		types.Err(http.StatusBadRequest, "hostname is required").Write(w)
		return
	}

	if err := v1Mutate(r, func(c *config.Config) error { c.Hostname = body.Hostname; return nil }); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, body)
}
