package kea

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	endpoints  map[string]string
	unix       bool
	useService bool
	Auth       string
	HTTP       *http.Client
}

func New(url, user, password string) *Client {
	c := &Client{
		endpoints:  map[string]string{"dhcp4": url, "dhcp6": url},
		useService: true,
		HTTP:       &http.Client{Timeout: 10 * time.Second},
	}
	if user != "" {
		c.Auth = basicAuth(user, password)
	}

	return c
}

func newLocal() *Client {
	return &Client{
		unix: true,
		endpoints: map[string]string{
			"dhcp4": LocalSock4,
			"dhcp6": LocalSock6,
		},
	}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}

	return http.DefaultClient
}

func (c *Client) send(service, command string, args any) (json.RawMessage, error) {
	endpoint := c.endpoints[service]
	if endpoint == "" {
		return nil, fmt.Errorf("no kea endpoint for service %q", service)
	}

	cmd := commandBase{Command: command}
	if c.useService {
		cmd.Service = []string{service}
	}

	if args != nil {
		raw, err := json.Marshal(args)
		if err != nil {
			return nil, err
		}

		cmd.Arguments = raw
	}

	body, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	var raw []byte
	if c.unix {
		raw, err = c.exchangeUnix(endpoint, body)
	} else {
		raw, err = c.exchangeHTTP(endpoint, body)
	}

	if err != nil {
		return nil, err
	}

	return parseResponse(command, raw)
}

func (c *Client) exchangeUnix(path string, body []byte) ([]byte, error) {
	conn, err := net.DialTimeout("unix", path, 5*time.Second)
	if err != nil {
		return nil, err
	}

	defer func(action func() error) { _ = action() }(conn.Close)

	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	if _, err := conn.Write(body); err != nil {
		return nil, err
	}

	if uc, ok := conn.(*net.UnixConn); ok {
		_ = uc.CloseWrite()
	}

	return io.ReadAll(conn)
}

func (c *Client) exchangeHTTP(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Auth != "" {
		req.Header.Set("Authorization", c.Auth)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}

	defer func(action func() error) { _ = action() }(resp.Body.Close)

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kea returned %s: %s", resp.Status, bytes.TrimSpace(raw))
	}

	return raw, nil
}

func parseResponse(command string, raw []byte) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty kea response")
	}

	var item response
	if trimmed[0] == '[' {
		var items []response
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, fmt.Errorf("decode kea response: %w", err)
		}

		if len(items) == 0 {
			return nil, fmt.Errorf("empty kea response")
		}

		item = items[0]
	} else if err := json.Unmarshal(trimmed, &item); err != nil {
		return nil, fmt.Errorf("decode kea response: %w", err)
	}

	if item.Result == 3 {
		switch command {
		case "lease4-get-all", "lease6-get-all":
			return json.RawMessage(`{"leases":[]}`), nil
		case "lease4-get", "lease6-get":
			return json.RawMessage(`{}`), nil
		}
	}

	if item.Result != 0 {
		if item.Text != "" {
			return nil, fmt.Errorf("%s", item.Text)
		}

		return nil, fmt.Errorf("kea command %q failed with result %d", command, item.Result)
	}

	return item.Arguments, nil
}

func (c *Client) getSubnetID(service, network string) (int, error) {
	subnets, err := c.configSubnets(service)
	if err != nil {
		return 0, err
	}

	for _, s := range subnets {
		if s.Subnet == network {
			return s.ID, nil
		}
	}

	return 0, fmt.Errorf("subnet %s not found", network)
}

func (c *Client) findSubnetForIP(service, ip string) (int, string, error) {
	subnets, err := c.configSubnets(service)
	if err != nil {
		return 0, "", err
	}

	for _, s := range subnets {
		if ipInCIDR(ip, s.Subnet) {
			return s.ID, s.Subnet, nil
		}
	}

	return 0, "", fmt.Errorf("no subnet found for %s", ip)
}

func (c *Client) resolve(input string) (service string, subnetID int, hostIP, network string, err error) {
	network, hostIP = parseCIDR(input)
	if network != "" {
		service = ipFamily(network)
		subnetID, err = c.getSubnetID(service, network)
		return service, subnetID, hostIP, network, err
	}

	service = ipFamily(hostIP)
	subnetID, network, err = c.findSubnetForIP(service, hostIP)
	return service, subnetID, hostIP, network, err
}

func (c *Client) Subnets(service string) ([]Subnet, error) {
	subnets, err := c.configSubnets(service)
	if err != nil {
		return nil, err
	}

	out := make([]Subnet, 0, len(subnets))
	for _, s := range subnets {
		out = append(out, Subnet{ID: s.ID, Subnet: s.Subnet})
	}

	return out, nil
}

func (c *Client) Leases(service string, subnetID int) ([]Lease, error) {
	leaseCmd := "lease4-get-all"
	if service == "dhcp6" {
		leaseCmd = "lease6-get-all"
	}

	raw, err := c.send(service, leaseCmd, reqLeaseSubnet{Subnets: []int{subnetID}})
	if err != nil {
		return nil, err
	}

	var result leaseList
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	return result.Leases, nil
}

func (c *Client) ConfigReload(service string) error {
	_, err := c.send(service, "config-reload", nil)
	return err
}

func (c *Client) AllLeases(service string) ([]Lease, error) {
	leaseCmd := "lease4-get-all"
	if service == "dhcp6" {
		leaseCmd = "lease6-get-all"
	}

	raw, err := c.send(service, leaseCmd, nil)
	if err != nil {
		return nil, err
	}

	var result leaseList
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	return result.Leases, nil
}

func (c *Client) LeaseByIP(ip string) (*Lease, error) {
	service := ipFamily(ip)
	leaseCmd := "lease4-get"
	if service == "dhcp6" {
		leaseCmd = "lease6-get"
	}

	raw, err := c.send(service, leaseCmd, reqIP{IPAddress: ip})
	if err != nil {
		if strings.Contains(err.Error(), "not find") || strings.Contains(err.Error(), "no lease") {
			return nil, nil
		}

		return nil, err
	}

	var lease Lease
	if err := json.Unmarshal(raw, &lease); err != nil {
		return nil, err
	}

	if lease.IPAddress == "" {
		return nil, nil
	}

	return &lease, nil
}

func (c *Client) Reservations(service string, subnetID int) ([]Host, error) {
	raw, err := c.send(service, "reservation-get-all", reqSubnet{SubnetID: subnetID})
	if err != nil {
		return nil, err
	}

	var result hostList
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}

	return result.Hosts, nil
}

func (c *Client) usedIPs(service string, subnetID int) []string {
	leases, err := c.Leases(service, subnetID)
	if err != nil {
		return nil
	}

	var ips []string
	for _, l := range leases {
		if l.IPAddress != "" {
			ips = append(ips, l.IPAddress)
		}
	}

	return ips
}

func (c *Client) reservedIPs(service string, subnetID int) []string {
	hosts, err := c.Reservations(service, subnetID)
	if err != nil {
		return nil
	}

	var ips []string
	for _, h := range hosts {
		if h.IPAddress != "" {
			ips = append(ips, h.IPAddress)
		}

		for _, a := range h.IPAddresses {
			if a != "" {
				ips = append(ips, a)
			}
		}
	}

	return ips
}

func (c *Client) ClearLease(input string) error {
	service, _, ip, _, err := c.resolve(input)
	if err != nil {
		return err
	}

	leaseCmd := "lease4-del"
	if service == "dhcp6" {
		leaseCmd = "lease6-del"
	}

	_, err = c.send(service, leaseCmd, reqIP{IPAddress: ip})
	return err
}

func (c *Client) AddReservation(hostname, identifier, input string) (string, error) {
	service, subnetID, ip, network, err := c.resolve(input)
	if err != nil {
		return "", err
	}

	used := c.usedIPs(service, subnetID)
	for _, r := range c.reservedIPs(service, subnetID) {
		if !contains(used, r) {
			used = append(used, r)
		}
	}

	if ip == "" {
		ip, err = nextAvailableIP(network, used)
		if err != nil {
			return "", err
		}
	} else if contains(used, ip) {
		return "", fmt.Errorf("IP %s is already in use", ip)
	}

	if service == "dhcp4" {
		res := reservationV4{SubnetID: subnetID, Hostname: hostname, IPAddress: ip}
		if isMAC(identifier) {
			res.HWAddress = identifier
		} else {
			res.DUID = identifier
		}

		if _, err := c.send(service, "reservation-add", reqReservationV4{Reservation: res}); err != nil {
			return "", err
		}

		return ip, nil
	}

	res := reservationV6{SubnetID: subnetID, Hostname: hostname, IPAddresses: []string{ip}, DUID: identifier}
	if _, err := c.send(service, "reservation-add", reqReservationV6{Reservation: res}); err != nil {
		return "", err
	}

	return ip, nil
}

func (c *Client) DelReservation(input string) error {
	service, subnetID, ip, _, err := c.resolve(input)
	if err != nil {
		return err
	}

	_, err = c.send(service, "reservation-del", reqSubnetIP{SubnetID: subnetID, IPAddress: ip})
	return err
}

func (c *Client) PersistLease(input string) (ip, network string, err error) {
	service, subnetID, ip, network, err := c.resolve(input)
	if err != nil {
		return "", "", err
	}

	if ip == "" {
		return "", "", fmt.Errorf("persist requires an IP, not just a network")
	}

	leases, err := c.Leases(service, subnetID)
	if err != nil {
		return "", "", err
	}

	var lease Lease
	found := false
	for _, l := range leases {
		if l.IPAddress == ip {
			lease = l
			found = true
			break
		}
	}

	if !found {
		return "", "", fmt.Errorf("no active lease found for %s", ip)
	}

	if contains(c.reservedIPs(service, subnetID), ip) {
		return "", "", fmt.Errorf("%s is already reserved", ip)
	}

	if service == "dhcp4" {
		res := reservationV4{SubnetID: subnetID, Hostname: lease.Hostname, IPAddress: ip}
		if lease.HWAddress != "" {
			res.HWAddress = lease.HWAddress
		} else {
			res.DUID = lease.DUID
		}

		if _, err := c.send(service, "reservation-add", reqReservationV4{Reservation: res}); err != nil {
			return "", "", err
		}

		return ip, network, nil
	}

	if lease.DUID == "" {
		return "", "", fmt.Errorf("no duid found in lease for %s", ip)
	}

	res := reservationV6{SubnetID: subnetID, Hostname: lease.Hostname, IPAddresses: []string{ip}, DUID: lease.DUID}
	if _, err := c.send(service, "reservation-add", reqReservationV6{Reservation: res}); err != nil {
		return "", "", err
	}

	return ip, network, nil
}

func contains(s []string, v string) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}

	return false
}
