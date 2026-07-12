package kea

import "encoding/json"

type cfgSubnet struct {
	ID           int       `json:"id"`
	Subnet       string    `json:"subnet"`
	Pools        []cfgPool `json:"pools"`
	Reservations []Host    `json:"reservations"`
}

type cfgPool struct {
	Pool string `json:"pool"`
}

type dhcpConfigRoot struct {
	Subnet4 []cfgSubnet `json:"subnet4"`
	Subnet6 []cfgSubnet `json:"subnet6"`
}

func rootKey(service string) string {
	if service == "dhcp6" {
		return "Dhcp6"
	}

	return "Dhcp4"
}

func (c *Client) configSubnets(service string) ([]cfgSubnet, error) {
	raw, err := c.send(service, "config-get", nil)
	if err != nil {
		return nil, err
	}

	var wrap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}

	var root dhcpConfigRoot
	if body, ok := wrap[rootKey(service)]; ok {
		if err := json.Unmarshal(body, &root); err != nil {
			return nil, err
		}
	}

	if service == "dhcp6" {
		return root.Subnet6, nil
	}

	return root.Subnet4, nil
}

func (c *Client) ReservedIPs() map[string]bool {
	set := make(map[string]bool)
	for _, service := range Services {
		subnets, err := c.configSubnets(service)
		if err != nil {
			continue
		}

		for _, s := range subnets {
			for _, ip := range s.reservedIPs() {
				set[ip] = true
			}
		}
	}

	return set
}

func (c *Client) FreeReservation(input string, exclusions []string) (subnet, ip string, err error) {
	return c.freeReservationIP(input, exclusions)
}

func (s cfgSubnet) reservedIPs() []string {
	var ips []string
	for _, h := range s.Reservations {
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
