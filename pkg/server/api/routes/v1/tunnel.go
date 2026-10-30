package v1

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func handleV1ReplaceTunnels(w http.ResponseWriter, r *http.Request) {
	body, err := v1Decode[map[string]*config.Tunnel](r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := v1Mutate(r, func(c *config.Config) error { return replaceSessionTunnels(c, body) }); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, body)
}

func replaceSessionTunnels(c *config.Config, tunnels map[string]*config.Tunnel) error {
	if err := c.ReplaceTunnels(tunnels); err != nil {
		return types.Wrap(http.StatusBadRequest, err, "invalid tunnels")
	}

	return nil
}

func handleV1SetTunnel(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	body, err := v1Decode[*config.Tunnel](r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := v1Mutate(r, func(c *config.Config) error { return setSessionTunnel(c, name, body) }); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, body)
}

func setSessionTunnel(c *config.Config, name string, tunnel *config.Tunnel) error {
	tunnels := c.Tunnels()
	tunnels[name] = tunnel
	return replaceSessionTunnels(c, tunnels)
}

func handleV1RemoveTunnel(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := v1Mutate(r, func(c *config.Config) error { return removeSessionTunnel(c, name) }); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, types.StatusResponse{Status: "deleted"})
}

func removeSessionTunnel(c *config.Config, name string) error {
	iface := c.Interfaces[name]
	if iface == nil || iface.Type != "tunnel" {
		return types.Errorf(http.StatusNotFound, "%q not found", name)
	}

	delete(c.Interfaces, name)
	return nil
}
