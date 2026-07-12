package dhcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/kea"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const maxRequestBodySize = 1 << 20

func clientFor(r *http.Request) (*kea.Client, error) {
	app := appctx.FromContext(r.Context())
	cfg, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context()))
	if err != nil {
		return nil, types.Wrap(http.StatusInternalServerError, err, "failed to read config")
	}

	url, user, password := "", "", ""
	if cfg.DHCP != nil && cfg.DHCP.ControlAgent != nil {
		ca := cfg.DHCP.ControlAgent
		url, user, password = ca.URL, ca.User, ca.Password
	}

	client, err := kea.Resolve(url, user, password)
	if err != nil {
		return nil, types.Wrap(http.StatusServiceUnavailable, err, "no kea control agent available")
	}

	return client, nil
}

func configFor(r *http.Request) *config.Config {
	app := appctx.FromContext(r.Context())
	cfg, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context()))
	if err != nil {
		return nil
	}

	return cfg
}

func parseTargetAddr(target string) (netip.Addr, error) {
	if strings.Contains(target, "/") {
		p, err := netip.ParsePrefix(target)
		if err != nil {
			return netip.Addr{}, err
		}

		return p.Addr(), nil
	}

	return netip.ParseAddr(target)
}

func subnetExclusions(cfg *config.Config, target string) []string {
	if cfg == nil || cfg.DHCP == nil {
		return nil
	}

	addr, err := parseTargetAddr(target)
	if err != nil {
		return nil
	}

	for _, list := range [][]config.KeaSubnet{cfg.DHCP.Subnets4, cfg.DHCP.Subnets6} {
		for _, s := range list {
			if prefix, perr := netip.ParsePrefix(s.Subnet); perr == nil && prefix.Contains(addr) {
				return s.Exclusions
			}
		}
	}

	return nil
}

type targetRequest struct {
	Target string `json:"target"`
}

type reservationRequest struct {
	Hostname   string `json:"hostname"`
	Identifier string `json:"identifier"`
	Target     string `json:"target"`
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		types.Err(http.StatusBadRequest, "failed to read request body").Write(w)
		return false
	}

	if err := json.Unmarshal(body, dst); err != nil {
		types.Error(log.Logger, w, types.Errorf(http.StatusBadRequest, "invalid request: %v", err))
		return false
	}

	return true
}

// Overview godoc
// @Summary  DHCP subnets with their leases and reservations
// @Tags dhcp
// @Produce json
// @Success 200 {object} types.Response[[]kea.SubnetView]
// @Security BearerAuth
// @Router /api/dhcp [get]
func Overview(w http.ResponseWriter, r *http.Request) {
	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	views, err := client.Overview()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to query kea"))
		return
	}

	if views == nil {
		views = []kea.SubnetView{}
	}

	types.OK(w, views)
}

// Subnets godoc
// @Summary  List configured DHCP subnets
// @Tags dhcp
// @Produce json
// @Success 200 {object} types.Response[[]kea.Subnet]
// @Security BearerAuth
// @Router /api/dhcp/subnets [get]
func Subnets(w http.ResponseWriter, r *http.Request) {
	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	var all []kea.Subnet
	for _, service := range kea.Services {
		subnets, err := client.Subnets(service)
		if err != nil {
			continue
		}

		all = append(all, subnets...)
	}

	if all == nil {
		all = []kea.Subnet{}
	}

	types.OK(w, all)
}

// AddReservation godoc
// @Summary  Reserve an address for a MAC/DUID
// @Tags dhcp
// @Produce json
// @Param body body dhcp.reservationRequest true "reservation"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/dhcp/reservations [post]
func AddReservation(w http.ResponseWriter, r *http.Request) {
	var req reservationRequest
	if !decode(w, r, &req) {
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	ip, err := client.AddReservation(req.Hostname, req.Identifier, req.Target)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to add reservation"))
		return
	}

	types.OK(w, types.StatusResponse{Status: ip})
}

// DelReservation godoc
// @Summary  Delete a reservation
// @Tags dhcp
// @Produce json
// @Param body body dhcp.targetRequest true "target ip or cidr"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/dhcp/reservations/del [post]
func DelReservation(w http.ResponseWriter, r *http.Request) {
	var req targetRequest
	if !decode(w, r, &req) {
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := client.DelReservation(req.Target); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to delete reservation"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}

// PersistLease godoc
// @Summary  Promote an active lease to a reservation
// @Tags dhcp
// @Produce json
// @Param body body dhcp.targetRequest true "target ip or cidr"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/dhcp/reservations/persist [post]
func PersistLease(w http.ResponseWriter, r *http.Request) {
	var req targetRequest
	if !decode(w, r, &req) {
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	ip, _, err := client.PersistLease(req.Target)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to persist lease"))
		return
	}

	types.OK(w, types.StatusResponse{Status: ip})
}

// ClearLease godoc
// @Summary  Delete an active lease
// @Tags dhcp
// @Produce json
// @Param body body dhcp.targetRequest true "target ip or cidr"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/dhcp/leases/clear [post]
func ClearLease(w http.ResponseWriter, r *http.Request) {
	var req targetRequest
	if !decode(w, r, &req) {
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := client.ClearLease(req.Target); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to clear lease"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
