package v1

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
)

func sliceOrEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}

	return s
}

// @Summary  Get session nftables
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/nftables [get]
func handleV1GetNftables(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) *config.NftablesConfig { return c.Nftables })
}

// @Summary  Replace session nftables
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/nftables [put]
func handleV1PutNftables(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v *config.NftablesConfig) { c.Nftables = v })
}

// @Summary  Get session boot modules
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {array} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/boot_modules [get]
func handleV1GetBootModules(w http.ResponseWriter, r *http.Request) {
	v1GetObject(w, r, func(c *config.Config) []string { return sliceOrEmpty(c.BootModules) })
}

// @Summary  Replace session boot modules
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {array} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/boot_modules [put]
func handleV1PutBootModules(w http.ResponseWriter, r *http.Request) {
	v1PutObject(w, r, func(c *config.Config, v []string) { c.BootModules = v })
}
