package dns

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/daemon/bind"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type ddnsZoneView struct {
	Name    string            `json:"name"`
	Records []bind.ZoneRecord `json:"records,omitempty" validate:"optional"`
	Error   string            `json:"error,omitempty" validate:"optional"`
}

type ddnsDeleteRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value,omitempty" validate:"optional"`
}

// @Summary  Live DDNS-managed zones and the dynamic records they currently hold
// @Tags dns
// @Produce json
// @Success 200 {object} types.Response[[]dns.ddnsZoneView]
// @Security BearerAuth
// @Router /api/dns/ddns [get]
func DDNS(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := serverFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if !render.DDNSActive(cfg) {
		types.OK(w, []ddnsZoneView{})
		return
	}

	client, err := ddnsClient(r, cfg)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	names := render.DDNSZoneNames(cfg)
	views := make([]ddnsZoneView, 0, len(names))
	for _, name := range names {
		zone := config.NormalizeDNSName(name)
		view := ddnsZoneView{Name: strings.TrimSuffix(zone, ".")}

		records, err := client.DynamicRecords(zone)
		if err != nil {
			view.Error = err.Error()
		} else {
			view.Records = records
		}

		views = append(views, view)
	}

	types.OK(w, views)
}

// @Summary  Clear a single dynamic record from a DDNS-managed zone
// @Tags dns
// @Accept json
// @Produce json
// @Param zone path string true "zone name"
// @Param request body dns.ddnsDeleteRequest true "record to delete"
// @Success 200 {object} types.Response[string]
// @Security BearerAuth
// @Router /api/dns/ddns/{zone}/records [delete]
func DeleteDDNSRecord(w http.ResponseWriter, r *http.Request) {
	cfg, _, err := serverFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	zone := chi.URLParam(r, "zone")
	if !render.DDNSActive(cfg) || !isDDNSZone(cfg, zone) {
		types.Error(log.Logger, w, types.NewError(http.StatusNotFound, fmt.Sprintf("zone %q is not DDNS-managed", zone)))
		return
	}

	var req ddnsDeleteRequest
	if err := decodeBody(r, &req); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if req.Name == "" || req.Type == "" {
		types.Error(log.Logger, w, types.NewError(http.StatusBadRequest, "name and type are required"))
		return
	}

	client, err := ddnsClient(r, cfg)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := client.DeleteRecord(config.NormalizeDNSName(zone), config.NormalizeDNSName(req.Name), req.Type, req.Value); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to delete the record"))
		return
	}

	types.OK(w, "deleted")
}

func ddnsClient(r *http.Request, cfg *config.Config) (*bind.Client, error) {
	client, err := clientFor(r)
	if err != nil {
		return nil, err
	}

	configureResolverTarget(client, cfg, cfg.DNS.Server)

	client.TSIGName = render.DDNSKeyName
	client.TSIGAlgorithm = cfg.DHCP.DDNS.Algorithm
	if client.TSIGAlgorithm == "" {
		client.TSIGAlgorithm = config.DefaultDDNSAlgorithm
	}

	client.TSIGSecret = cfg.DHCP.DDNS.Key

	return client, nil
}

func isDDNSZone(cfg *config.Config, zone string) bool {
	want := config.NormalizeDNSName(zone)
	for _, name := range render.DDNSZoneNames(cfg) {
		if config.NormalizeDNSName(name) == want {
			return true
		}
	}

	return false
}
