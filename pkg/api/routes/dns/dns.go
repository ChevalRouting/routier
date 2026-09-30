package dns

import (
	"encoding/json"
	"io"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/bind"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const maxRequestBodySize = 1 << 20

func configFor(r *http.Request) (*config.Config, error) {
	app := appctx.FromContext(r.Context())

	cfg, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context()))
	if err != nil {
		return nil, types.Wrap(http.StatusInternalServerError, err, "failed to read config")
	}

	return cfg, nil
}

func serverFor(r *http.Request) (*config.Config, *config.DNSServer, error) {
	cfg, err := configFor(r)
	if err != nil {
		return nil, nil, err
	}

	if cfg.DNS == nil || cfg.DNS.Server == nil || !cfg.DNS.Server.Enabled {
		return cfg, nil, types.NewError(http.StatusNotFound, "no dns server is configured")
	}

	return cfg, cfg.DNS.Server, nil
}

func clientFor(r *http.Request) (*bind.Client, error) {
	client, err := bind.LoadLocal()
	if err != nil {
		return nil, types.Wrap(http.StatusServiceUnavailable, err, "rndc control channel unavailable")
	}

	return client, nil
}

func configureResolverTarget(client *bind.Client, cfg *config.Config, s *config.DNSServer) {
	client.Resolver = render.DNSQueryAddress(cfg)
	if s.Port > 0 {
		client.ResolverPort = s.Port
	}
}

func zoneInputs(s *config.DNSServer) []bind.ZoneInput {
	out := make([]bind.ZoneInput, 0, len(s.Zones))
	for _, z := range s.Zones {
		in := bind.ZoneInput{Name: z.Name}
		if z.SOA != nil {
			in.Serial = z.SOA.Serial
		}

		for _, rec := range z.Records {
			in.Records = append(in.Records, bind.ZoneRecord{
				Name:     rec.Name,
				Type:     rec.Type,
				Value:    rec.Value,
				TTL:      rec.TTL,
				Priority: rec.Priority,
			})
		}

		out = append(out, in)
	}

	return out
}

func decodeBody(r *http.Request, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		return types.Wrap(http.StatusBadRequest, err, "failed to read request body")
	}

	if len(body) == 0 {
		return nil
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return types.Wrap(http.StatusBadRequest, err, "invalid request body")
	}

	return nil
}

type overviewResponse struct {
	Mode      string         `json:"mode"`
	Recurses  bool           `json:"recurses"`
	Port      int            `json:"port"`
	AllowFrom []string       `json:"allow_from,omitempty" validate:"optional"`
	Upstreams []string       `json:"upstreams,omitempty" validate:"optional"`
	Overview  *bind.Overview `json:"overview,omitempty" validate:"optional"`
}

// Overview godoc
// @Summary  Resolver state, listen addresses, forward zones and authoritative zones
// @Tags dns
// @Produce json
// @Success 200 {object} types.Response[dns.overviewResponse]
// @Security BearerAuth
// @Router /api/dns [get]
func Overview(w http.ResponseWriter, r *http.Request) {
	cfg, s, err := serverFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	port := s.Port
	if port == 0 {
		port = bind.DefaultPort
	}

	resp := overviewResponse{
		Mode:      s.ResolvedMode(),
		Recurses:  s.Recurses(),
		Port:      port,
		AllowFrom: s.AllowFrom,
		Upstreams: s.Upstreams,
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	configureResolverTarget(client, cfg, s)

	in := bind.OverviewInput{Listen: render.DNSListenAddresses(cfg), Zones: zoneInputs(s)}

	view, err := client.Overview(in)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to query named"))
		return
	}

	resp.Overview = view

	types.OK(w, resp)
}
