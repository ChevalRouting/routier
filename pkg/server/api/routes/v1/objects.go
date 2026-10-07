package v1

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
)

// @Summary  Get session dns
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/dns [get]
func handleV1GetDNS(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) *config.DNS { return c.DNS })
}

// @Summary  Replace session dns
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/dns [put]
func handleV1PutDNS(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v *config.DNS) { c.DNS = v })
}

// @Summary  Get session routing
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/routing [get]
func handleV1GetRouting(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) *config.Routing { return c.Routing })
}

// @Summary  Replace session routing
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/routing [put]
func handleV1PutRouting(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v *config.Routing) { c.Routing = v })
}

// @Summary  Get session logging
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/logging [get]
func handleV1GetLogging(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) *config.Logging { return c.Logging })
}

// @Summary  Replace session logging
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/logging [put]
func handleV1PutLogging(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v *config.Logging) { c.Logging = v })
}

// @Summary  Get session ssh
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/ssh [get]
func handleV1GetSSH(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) *config.SSH { return c.SSH })
}

// @Summary  Replace session ssh
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/ssh [put]
func handleV1PutSSH(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v *config.SSH) { c.SSH = v })
}

// @Summary  Get session HA config (vrrp, conntrackd)
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/ha [get]
func handleV1GetHA(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) *config.HA { return c.HA })
}

// @Summary  Replace session HA config (vrrp, conntrackd)
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/ha [put]
func handleV1PutHA(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v *config.HA) { c.HA = v })
}

// @Summary  Get session gai
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/gai [get]
func handleV1GetGAI(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) *config.GAIConfig { return c.GAI })
}

// @Summary  Replace session gai
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/gai [put]
func handleV1PutGAI(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v *config.GAIConfig) { c.GAI = v })
}

// @Summary  Get session monitoring
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/monitoring [get]
func handleV1GetMonitoring(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) *config.MonitoringConfig { return c.Monitoring })
}

// @Summary  Replace session monitoring
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/monitoring [put]
func handleV1PutMonitoring(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v *config.MonitoringConfig) { c.Monitoring = v })
}
