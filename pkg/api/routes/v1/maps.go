package v1

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
)

// handleV1GetInterfaces godoc
// @Summary  List session interfaces
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/interfaces [get]
func handleV1GetInterfaces(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) map[string]*config.Interface { return mapOrEmpty(c.Interfaces) })
}

// handleV1PutInterfaces godoc
// @Summary  Replace session interfaces
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/interfaces [put]
func handleV1PutInterfaces(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v map[string]*config.Interface) { c.Interfaces = v })
}

// handleV1GetInterface godoc
// @Summary  Get one Interface
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/interfaces/{name} [get]
func handleV1GetInterface(w http.ResponseWriter, r *http.Request) {
	v1MapGetKey(w, r, func(c *config.Config) map[string]*config.Interface { return c.Interfaces })
}

// handleV1PutInterface godoc
// @Summary  Set one Interface
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/interfaces/{name} [put]
func handleV1PutInterface(w http.ResponseWriter, r *http.Request) {
	v1MapPutKey(w, r,
		func(c *config.Config) map[string]*config.Interface { return c.Interfaces },
		func(c *config.Config, m map[string]*config.Interface) { c.Interfaces = m })
}

// handleV1DeleteInterface godoc
// @Summary  Delete one Interface
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/interfaces/{name} [delete]
func handleV1DeleteInterface(w http.ResponseWriter, r *http.Request) {
	v1MapDeleteKey(w, r, func(c *config.Config) map[string]*config.Interface { return c.Interfaces })
}

// handleV1GetTunnels godoc
// @Summary  List session tunnels
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/tunnels [get]
func handleV1GetTunnels(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) map[string]*config.Tunnel { return mapOrEmpty(c.Tunnels) })
}

// handleV1PutTunnels godoc
// @Summary  Replace session tunnels
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/tunnels [put]
func handleV1PutTunnels(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v map[string]*config.Tunnel) { c.Tunnels = v })
}

// handleV1GetTunnel godoc
// @Summary  Get one Tunnel
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/tunnels/{name} [get]
func handleV1GetTunnel(w http.ResponseWriter, r *http.Request) {
	v1MapGetKey(w, r, func(c *config.Config) map[string]*config.Tunnel { return c.Tunnels })
}

// handleV1PutTunnel godoc
// @Summary  Set one Tunnel
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/tunnels/{name} [put]
func handleV1PutTunnel(w http.ResponseWriter, r *http.Request) {
	v1MapPutKey(w, r,
		func(c *config.Config) map[string]*config.Tunnel { return c.Tunnels },
		func(c *config.Config, m map[string]*config.Tunnel) { c.Tunnels = m })
}

// handleV1DeleteTunnel godoc
// @Summary  Delete one Tunnel
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/tunnels/{name} [delete]
func handleV1DeleteTunnel(w http.ResponseWriter, r *http.Request) {
	v1MapDeleteKey(w, r, func(c *config.Config) map[string]*config.Tunnel { return c.Tunnels })
}

// handleV1GetWireguards godoc
// @Summary  List session wireguard
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/wireguard [get]
func handleV1GetWireguards(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) map[string]*config.Wireguard { return mapOrEmpty(c.Wireguard) })
}

// handleV1PutWireguards godoc
// @Summary  Replace session wireguard
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/wireguard [put]
func handleV1PutWireguards(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v map[string]*config.Wireguard) { c.Wireguard = v })
}

// handleV1GetWireguard godoc
// @Summary  Get one Wireguard
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/wireguard/{name} [get]
func handleV1GetWireguard(w http.ResponseWriter, r *http.Request) {
	v1MapGetKey(w, r, func(c *config.Config) map[string]*config.Wireguard { return c.Wireguard })
}

// handleV1PutWireguard godoc
// @Summary  Set one Wireguard
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/wireguard/{name} [put]
func handleV1PutWireguard(w http.ResponseWriter, r *http.Request) {
	v1MapPutKey(w, r,
		func(c *config.Config) map[string]*config.Wireguard { return c.Wireguard },
		func(c *config.Config, m map[string]*config.Wireguard) { c.Wireguard = m })
}

// handleV1DeleteWireguard godoc
// @Summary  Delete one Wireguard
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/wireguard/{name} [delete]
func handleV1DeleteWireguard(w http.ResponseWriter, r *http.Request) {
	v1MapDeleteKey(w, r, func(c *config.Config) map[string]*config.Wireguard { return c.Wireguard })
}

// handleV1GetUsers godoc
// @Summary  List session users
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/users [get]
func handleV1GetUsers(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) map[string]*config.User { return mapOrEmpty(c.Users) })
}

// handleV1PutUsers godoc
// @Summary  Replace session users
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/users [put]
func handleV1PutUsers(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v map[string]*config.User) { c.Users = v })
}

// handleV1GetUser godoc
// @Summary  Get one User
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/users/{name} [get]
func handleV1GetUser(w http.ResponseWriter, r *http.Request) {
	v1MapGetKey(w, r, func(c *config.Config) map[string]*config.User { return c.Users })
}

// handleV1PutUser godoc
// @Summary  Set one User
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/users/{name} [put]
func handleV1PutUser(w http.ResponseWriter, r *http.Request) {
	v1MapPutKey(w, r,
		func(c *config.Config) map[string]*config.User { return c.Users },
		func(c *config.Config, m map[string]*config.User) { c.Users = m })
}

// handleV1DeleteUser godoc
// @Summary  Delete one User
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/users/{name} [delete]
func handleV1DeleteUser(w http.ResponseWriter, r *http.Request) {
	v1MapDeleteKey(w, r, func(c *config.Config) map[string]*config.User { return c.Users })
}

// handleV1GetServices godoc
// @Summary  List session services
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/services [get]
func handleV1GetServices(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) map[string]*config.Service { return mapOrEmpty(c.Services) })
}

// handleV1PutServices godoc
// @Summary  Replace session services
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/services [put]
func handleV1PutServices(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v map[string]*config.Service) { c.Services = v })
}

// handleV1GetService godoc
// @Summary  Get one Service
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/services/{name} [get]
func handleV1GetService(w http.ResponseWriter, r *http.Request) {
	v1MapGetKey(w, r, func(c *config.Config) map[string]*config.Service { return c.Services })
}

// handleV1PutService godoc
// @Summary  Set one Service
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/services/{name} [put]
func handleV1PutService(w http.ResponseWriter, r *http.Request) {
	v1MapPutKey(w, r,
		func(c *config.Config) map[string]*config.Service { return c.Services },
		func(c *config.Config, m map[string]*config.Service) { c.Services = m })
}

// handleV1DeleteService godoc
// @Summary  Delete one Service
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/services/{name} [delete]
func handleV1DeleteService(w http.ResponseWriter, r *http.Request) {
	v1MapDeleteKey(w, r, func(c *config.Config) map[string]*config.Service { return c.Services })
}

// handleV1GetVRFs godoc
// @Summary  List session vrfs
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/vrfs [get]
func handleV1GetVRFs(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) map[string]*config.VRFConfig { return mapOrEmpty(c.VRFs) })
}

// handleV1PutVRFs godoc
// @Summary  Replace session vrfs
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/vrfs [put]
func handleV1PutVRFs(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v map[string]*config.VRFConfig) { c.VRFs = v })
}

// handleV1GetVRF godoc
// @Summary  Get one VRF
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/vrfs/{name} [get]
func handleV1GetVRF(w http.ResponseWriter, r *http.Request) {
	v1MapGetKey(w, r, func(c *config.Config) map[string]*config.VRFConfig { return c.VRFs })
}

// handleV1PutVRF godoc
// @Summary  Set one VRF
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/vrfs/{name} [put]
func handleV1PutVRF(w http.ResponseWriter, r *http.Request) {
	v1MapPutKey(w, r,
		func(c *config.Config) map[string]*config.VRFConfig { return c.VRFs },
		func(c *config.Config, m map[string]*config.VRFConfig) { c.VRFs = m })
}

// handleV1DeleteVRF godoc
// @Summary  Delete one VRF
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param name path string true "key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/vrfs/{name} [delete]
func handleV1DeleteVRF(w http.ResponseWriter, r *http.Request) {
	v1MapDeleteKey(w, r, func(c *config.Config) map[string]*config.VRFConfig { return c.VRFs })
}
