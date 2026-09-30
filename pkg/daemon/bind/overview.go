package bind

type OverviewInput struct {
	Listen []string    `json:"listen,omitempty" validate:"optional"`
	Zones  []ZoneInput `json:"zones,omitempty"  validate:"optional"`
}

type Overview struct {
	Running bool       `json:"running"`
	Status  *Status    `json:"status,omitempty" validate:"optional"`
	Listen  []string   `json:"listen,omitempty" validate:"optional"`
	Zones   []ZoneView `json:"zones,omitempty"  validate:"optional"`
	Stats   []Stat     `json:"stats,omitempty"  validate:"optional"`
}

func (c *Client) Overview(in OverviewInput) (*Overview, error) {
	out := &Overview{Listen: in.Listen, Zones: DeclaredZones(in.Zones)}

	status, err := c.Status()
	if err != nil {
		return out, nil
	}

	out.Status = status
	out.Running = status.Running
	out.Zones = c.Zones(in.Zones)

	if stats, err := c.Statistics(); err == nil {
		out.Stats = KeyStatistics(stats)
	}

	return out, nil
}
