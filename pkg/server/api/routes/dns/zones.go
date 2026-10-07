package dns

import (
	"fmt"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/daemon/bind"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// @Summary  Authoritative zones with their declared records and live serial
// @Tags dns
// @Produce json
// @Success 200 {object} types.Response[[]bind.ZoneView]
// @Security BearerAuth
// @Router /api/dns/zones [get]
func Zones(w http.ResponseWriter, r *http.Request) {
	cfg, s, err := serverFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	inputs := zoneInputs(s)

	client, err := bind.LoadLocal()
	if err != nil {
		types.OK(w, bind.DeclaredZones(inputs))
		return
	}

	configureResolverTarget(client, cfg, s)

	types.OK(w, client.Zones(inputs))
}

// @Summary  A single authoritative zone
// @Tags dns
// @Produce json
// @Param name path string true "zone name"
// @Success 200 {object} types.Response[bind.ZoneView]
// @Security BearerAuth
// @Router /api/dns/zones/{name} [get]
func Zone(w http.ResponseWriter, r *http.Request) {
	cfg, s, err := serverFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	name := chi.URLParam(r, "name")

	input, ok := findZone(s, name)
	if !ok {
		types.Error(log.Logger, w, types.NewError(http.StatusNotFound, fmt.Sprintf("zone %q is not configured", name)))
		return
	}

	client, err := bind.LoadLocal()
	if err != nil {
		types.OK(w, bind.DeclaredZones([]bind.ZoneInput{input})[0])
		return
	}

	configureResolverTarget(client, cfg, s)

	types.OK(w, client.Zones([]bind.ZoneInput{input})[0])
}

// @Summary  Reload one authoritative zone from its rendered file
// @Tags dns
// @Produce json
// @Param name path string true "zone name"
// @Success 200 {object} types.Response[string]
// @Security BearerAuth
// @Router /api/dns/zones/{name}/reload [post]
func ReloadZone(w http.ResponseWriter, r *http.Request) {
	_, s, err := serverFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	name := chi.URLParam(r, "name")

	input, ok := findZone(s, name)
	if !ok {
		types.Error(log.Logger, w, types.NewError(http.StatusNotFound, fmt.Sprintf("zone %q is not configured", name)))
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := client.ReloadZone(input.Name); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to reload the zone"))
		return
	}

	types.OK(w, "reloaded")
}

func findZone(s *config.DNSServer, name string) (bind.ZoneInput, bool) {
	want := config.NormalizeDNSName(name)
	for _, in := range zoneInputs(s) {
		if config.NormalizeDNSName(in.Name) == want {
			return in, true
		}
	}

	return bind.ZoneInput{}, false
}
