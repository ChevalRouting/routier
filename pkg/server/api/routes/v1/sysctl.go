package v1

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type v1SysctlEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// @Summary  Get session sysctls
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/sysctl [get]
func handleV1GetSysctl(w http.ResponseWriter, r *http.Request) {
	sess := v1SessionFromCtx(r.Context())
	m := sess.Config.Sysctl
	if m == nil {
		m = map[string]string{}
	}

	types.OK(w, m)
}

// @Summary  Replace session sysctls
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/sysctl [put]
func handleV1PutSysctl(w http.ResponseWriter, r *http.Request) {
	body, err := v1Decode[map[string]string](r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := v1Mutate(r, func(c *config.Config) error { c.Sysctl = body; return nil }); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, body)
}

// @Summary  Get one sysctl
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param key path string true "sysctl key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/sysctl/{key} [get]
func handleV1GetSysctlKey(w http.ResponseWriter, r *http.Request) {
	sess := v1SessionFromCtx(r.Context())
	key := chi.URLParam(r, "key")
	val, ok := sess.Config.Sysctl[key]
	if !ok {
		types.Error(log.Logger, w, types.Errorf(http.StatusNotFound, "sysctl key %q not found", key))
		return
	}

	types.OK(w, v1SysctlEntry{Key: key, Value: val})
}

// @Summary  Set one sysctl
// @Tags v1-config
// @Accept json
// @Produce json
// @Param sessionID path string true "session id"
// @Param key path string true "sysctl key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/sysctl/{key} [put]
type putSysctlKeyBody struct {
	Value string `json:"value"`
}

func handleV1PutSysctlKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	body, err := v1Decode[putSysctlKeyBody](r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := v1Mutate(r, func(c *config.Config) error { return handleV1PutSysctlKeyCallback(key, &body, c) }); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, v1SysctlEntry{Key: key, Value: body.Value})
}

// @Summary  Delete one sysctl
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Param key path string true "sysctl key"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/sysctl/{key} [delete]
func handleV1DeleteSysctlKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if err := v1Mutate(r, func(c *config.Config) error { return handleV1DeleteSysctlKeyCallback(key, c) }); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, types.StatusResponse{Status: "deleted"})
}

func handleV1PutSysctlKeyCallback(key string, body *putSysctlKeyBody, c *config.Config) error {
	if c.Sysctl == nil {
		c.Sysctl = make(map[string]string)
	}

	c.Sysctl[key] = (*body).Value
	return nil
}

func handleV1DeleteSysctlKeyCallback(key string, c *config.Config) error {
	if _, exists := c.Sysctl[key]; !exists {
		return types.Errorf(http.StatusNotFound, "sysctl key %q not found", key)
	}

	delete(c.Sysctl, key)
	return nil
}
