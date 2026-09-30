package dhcp

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"os"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/iptools"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

type reserveFromLeaseRequest struct {
	LeaseIP  string `json:"lease_ip"`
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	DNSZone  string `json:"dns_zone,omitempty" validate:"optional"`
	DNSName  string `json:"dns_name,omitempty" validate:"optional"`
}

// ReserveFromLease adds a reservation for a leased host, applies it, then clears
// the lease so the client picks up its reserved address on its next request.
//
// @Summary  Reserve a host from a dynamic lease
// @Tags dhcp
// @Produce json
// @Param body body dhcp.reserveFromLeaseRequest true "lease ip and chosen reservation ip"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/dhcp/reservations/from-lease [post]
func ReserveFromLease(w http.ResponseWriter, r *http.Request) {
	var req reserveFromLeaseRequest
	if !decode(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.LeaseIP) == "" {
		types.Err(http.StatusBadRequest, "lease ip is required").Write(w)
		return
	}

	client, err := clientFor(r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	lease, err := client.LeaseByIP(req.LeaseIP)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to read lease"))
		return
	}

	if lease == nil {
		types.Err(http.StatusNotFound, "no active lease for "+req.LeaseIP).Write(w)
		return
	}

	subnetCIDR, err := client.SubnetCIDR(req.LeaseIP)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to resolve subnet"))
		return
	}

	resIP := strings.TrimSpace(req.IP)
	if resIP == "" {
		if _, resIP, err = client.FreeReservation(req.LeaseIP, subnetExclusions(configFor(r), req.LeaseIP)); err != nil {
			types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to find a free address"))
			return
		}
	}

	prefix, perr := netip.ParsePrefix(subnetCIDR)
	addr, aerr := netip.ParseAddr(resIP)
	if perr != nil || aerr != nil || !prefix.Contains(addr) {
		types.Err(http.StatusBadRequest, resIP+" is not a valid address in "+subnetCIDR).Write(w)
		return
	}

	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	live, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	if live.DHCP == nil {
		types.Err(http.StatusBadRequest, "DHCP is not configured").Write(w)
		return
	}

	hostname := strings.TrimSpace(req.Hostname)
	if hostname == "" {
		hostname = lease.Hostname
	}

	v6 := addr.Is6()
	res := config.KeaReservation{Hostname: hostname, IPAddress: resIP}
	if v6 {
		res.DUID = lease.DUID
	} else {
		res.HWAddress = lease.HWAddress
	}

	if !addReservationToSubnet(live, subnetCIDR, v6, res) {
		types.Err(http.StatusBadRequest, "subnet "+subnetCIDR+" is not in the DHCP config").Write(w)
		return
	}

	dnsZone := strings.TrimSpace(req.DNSZone)
	if dnsZone != "" {
		if _, err := addReservationDNS(live, dnsZone, req.DNSName, resIP); err != nil {
			types.Err(http.StatusBadRequest, err.Error()).Write(w)
			return
		}
	}

	if appErr := cfgstore.ValidationError(live); appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	resolved, err := cfgstore.Resolve(live)
	if err != nil {
		types.Err(http.StatusBadRequest, "config interpolation: "+err.Error()).Write(w)
		return
	}

	if _, err := managers.Apply(r.Context(), resolved, friendcache.InterpolationVars(),
		managers.ApplyOptions{Source: "web", ConfigPath: app.ConfigPath}, 0); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to apply reservation"))
		return
	}

	if err := cfgstore.PromoteConfig(app.ConfigPath, live); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "reservation applied but failed to persist config"))
		return
	}

	preserveStagedReservation(app.ConfigPath, username, subnetCIDR, v6, res, dnsZone, req.DNSName, resIP)

	_ = client.ClearLease(req.LeaseIP)

	types.OK(w, types.StatusResponse{Status: resIP})
}

func addReservationToSubnet(cfg *config.Config, subnetCIDR string, v6 bool, res config.KeaReservation) bool {
	if cfg.DHCP == nil {
		return false
	}

	subnets := cfg.DHCP.Subnets4
	if v6 {
		subnets = cfg.DHCP.Subnets6
	}

	for i := range subnets {
		if subnets[i].Subnet == subnetCIDR {
			subnets[i].Reservations = append(subnets[i].Reservations, res)
			return true
		}
	}

	return false
}

func preserveStagedReservation(configPath, username, subnetCIDR string, v6 bool, res config.KeaReservation, dnsZone, dnsName, resIP string) {
	stagingPath := cfgstore.StagingPath(configPath, username)
	if _, err := os.Stat(stagingPath); err != nil {
		return
	}

	staged, err := config.Load(stagingPath)
	if err != nil {
		return
	}

	changed := addReservationToSubnet(staged, subnetCIDR, v6, res)
	if dnsZone != "" {
		if _, err := addReservationDNS(staged, dnsZone, dnsName, resIP); err == nil {
			changed = true
		}
	}

	if changed {
		_ = cfgstore.WriteStaging(configPath, username, staged)
	}
}

func addReservationDNS(cfg *config.Config, zoneName, name, ip string) (string, error) {
	if cfg.DNS == nil || cfg.DNS.Server == nil {
		return "", errors.New("the DNS server is not configured")
	}

	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return "", fmt.Errorf("%q is not a valid address", ip)
	}

	server := cfg.DNS.Server
	zone := findAuthoritativeZone(server, zoneName)
	if zone == nil {
		return "", fmt.Errorf("no authoritative zone %q in the DNS config", zoneName)
	}

	label := dnsRecordLabel(name, zone.Name)
	if label == "" {
		return "", errors.New("a record name is required")
	}

	rtype := "A"
	if addr.Is6() {
		rtype = "AAAA"
	}

	zone.Records = append(zone.Records, config.DNSRecord{Name: label, Type: rtype, Value: addr.String()})

	fqdn := recordFQDN(label, zone.Name)
	addReversePTR(server, addr, fqdn)

	return fqdn, nil
}

func findAuthoritativeZone(s *config.DNSServer, name string) *config.DNSZone {
	want := config.NormalizeDNSName(name)
	for i := range s.Zones {
		z := &s.Zones[i]
		if len(z.Primaries) == 0 && config.NormalizeDNSName(z.Name) == want {
			return z
		}
	}

	return nil
}

func dnsRecordLabel(name, zone string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "@" {
		return name
	}

	origin := config.NormalizeDNSName(zone)
	full := config.NormalizeDNSName(name)
	if full == origin {
		return "@"
	}

	if strings.HasSuffix(full, "."+origin) {
		return strings.TrimSuffix(full, "."+origin)
	}

	return strings.TrimSuffix(name, ".")
}

func recordFQDN(label, zone string) string {
	origin := config.NormalizeDNSName(zone)
	if label == "@" {
		return origin
	}

	return label + "." + origin
}

func addReversePTR(s *config.DNSServer, addr netip.Addr, target string) {
	rev, err := iptools.Reverse(addr.String())
	if err != nil {
		return
	}

	full := config.NormalizeDNSName(rev.Name)

	var best *config.DNSZone
	var bestLen int
	for i := range s.Zones {
		z := &s.Zones[i]
		if len(z.Primaries) > 0 {
			continue
		}

		origin := config.NormalizeDNSName(z.Name)
		if !isReverseZone(origin) {
			continue
		}

		if (full == origin || strings.HasSuffix(full, "."+origin)) && len(origin) > bestLen {
			best, bestLen = z, len(origin)
		}
	}

	if best == nil {
		return
	}

	origin := config.NormalizeDNSName(best.Name)
	owner := "@"
	if full != origin {
		owner = strings.TrimSuffix(full, "."+origin)
	}

	best.Records = append(best.Records, config.DNSRecord{Name: owner, Type: "PTR", Value: target})
}

func isReverseZone(origin string) bool {
	return strings.HasSuffix(origin, "in-addr.arpa.") || strings.HasSuffix(origin, "ip6.arpa.")
}
