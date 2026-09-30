package kea

type SubnetView struct {
	Service      string  `json:"service"`
	ID           int     `json:"id"`
	Subnet       string  `json:"subnet"`
	Leases       []Lease `json:"leases,omitempty" validate:"optional"`
	Reservations []Host  `json:"reservations,omitempty" validate:"optional"`
}

var Services = []string{"dhcp4", "dhcp6"}

func (c *Client) Overview() ([]SubnetView, error) {
	var views []SubnetView
	var firstErr error

	for _, service := range Services {
		subnets, err := c.configSubnets(service)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}

			continue
		}

		for _, s := range subnets {
			view := SubnetView{Service: service, ID: s.ID, Subnet: s.Subnet, Reservations: s.Reservations}

			if leases, err := c.Leases(service, s.ID); err == nil {
				view.Leases = leases
			}

			views = append(views, view)
		}
	}

	if views == nil && firstErr != nil {
		return nil, firstErr
	}

	return views, nil
}
