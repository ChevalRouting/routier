package dhcp

import (
	"net"
	"net/http"
	"strings"

	"github.com/ChevalRouting/routier/pkg/daemon/kea"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

type leaseView struct {
	Service  string `json:"service"`
	Reserved bool   `json:"reserved"`
	kea.Lease
}

func leaseMatches(l kea.Lease, q string) bool {
	if q == "" {
		return true
	}

	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(l.IPAddress), q) ||
		strings.Contains(strings.ToLower(l.HWAddress), q) ||
		strings.Contains(strings.ToLower(l.DUID), q) ||
		strings.Contains(strings.ToLower(l.Hostname), q)
}

// @Summary  List active DHCP leases, optionally filtered
// @Tags dhcp
// @Produce json
// @Param q query string false "filter (exact IP is looked up via kea, otherwise substring match on ip/mac/duid/hostname)"
// @Success 200 {object} types.Response[[]dhcp.leaseView]
// @Security BearerAuth
// @Router /api/dhcp/leases [get]
func Leases(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	out := []leaseView{}
	reserved := client.ReservedIPs()

	if ip := net.ParseIP(q); ip != nil {
		service := "dhcp4"
		if ip.To4() == nil {
			service = "dhcp6"
		}

		l, err := client.LeaseByIP(q)
		if err != nil {
			types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to read DHCP lease"))
			return
		}

		if l != nil {
			out = append(out, leaseView{Service: service, Reserved: reserved[l.IPAddress], Lease: *l})
		}

		types.OK(w, out)
		return
	}

	var firstErr error
	successful := false
	for _, service := range kea.Services {
		leases, err := client.AllLeases(service)
		if err != nil {
			log.Warn().Err(err).Str("service", service).Msg("failed to read DHCP leases")
			if firstErr == nil {
				firstErr = err
			}

			continue
		}

		successful = true

		for _, l := range leases {
			if leaseMatches(l, q) {
				out = append(out, leaseView{Service: service, Reserved: reserved[l.IPAddress], Lease: l})
			}
		}
	}

	if !successful && firstErr != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, firstErr, "failed to read DHCP leases"))
		return
	}

	types.OK(w, out)
}

// @Summary  Suggest a free address in a subnet's pools
// @Tags dhcp
// @Produce json
// @Param target query string true "subnet cidr or an address inside it"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/dhcp/free-ip [get]
func FreeIP(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	if target == "" {
		types.Err(http.StatusBadRequest, "target subnet is required").Write(w)
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	ip, err := client.FreeIP(target, subnetExclusions(configFor(r), target))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to find a free address"))
		return
	}

	types.OK(w, types.StatusResponse{Status: ip})
}
