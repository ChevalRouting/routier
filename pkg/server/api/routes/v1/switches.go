package v1

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
)

// @Summary  List session switches
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/switches [get]
func handleV1GetSwitches(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) map[string]*config.Switch { return mapOrEmpty(c.Switches) })
}

// @Summary  Replace session switches
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/switches [put]
func handleV1PutSwitches(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v map[string]*config.Switch) { c.Switches = v })
}

// @Summary  Get one switch
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/switches/{name} [get]
func handleV1GetSwitch(w http.ResponseWriter, r *http.Request) {
	v1MapGetKey(w, r, func(c *config.Config) map[string]*config.Switch { return c.Switches })
}

// @Summary  Set one switch
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/switches/{name} [put]
func handleV1PutSwitch(w http.ResponseWriter, r *http.Request) {
	v1MapPutKey(w, r,
		func(c *config.Config) map[string]*config.Switch { return c.Switches },
		func(c *config.Config, m map[string]*config.Switch) { c.Switches = m })
}

// @Summary  Delete one switch
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/switches/{name} [delete]
func handleV1DeleteSwitch(w http.ResponseWriter, r *http.Request) {
	v1MapDeleteKey(w, r, func(c *config.Config) map[string]*config.Switch { return c.Switches })
}
