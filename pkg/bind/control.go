package bind

import "strings"

type Status struct {
	Version       string `json:"version,omitempty"        validate:"optional"`
	BootTime      string `json:"boot_time,omitempty"      validate:"optional"`
	ConfigTime    string `json:"config_time,omitempty"    validate:"optional"`
	Zones         int    `json:"zones"`
	RecursiveHigh int    `json:"recursive_high_water"`
	Running       bool   `json:"running"`
}

func (c *Client) Status() (*Status, error) {
	out, err := c.rndc("status")
	if err != nil {
		return nil, err
	}

	return parseStatus(out), nil
}

func parseStatus(out string) *Status {
	status := &Status{Running: true}

	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
		if !ok {
			continue
		}

		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "version":
			status.Version = value
		case "boot time":
			status.BootTime = value
		case "last configured":
			status.ConfigTime = value
		case "number of zones":
			status.Zones = atoi(strings.Fields(value + " ")[0])
		case "recursive clients":
			if _, high, ok := strings.Cut(value, "/"); ok {
				status.RecursiveHigh = atoi(firstLine(high))
			}
		}
	}

	return status
}

func (c *Client) Reconfig() error {
	_, err := c.rndc("reconfig")

	return err
}

func (c *Client) ReloadAll() error {
	_, err := c.rndc("reload")

	return err
}

func (c *Client) ReloadZone(zone string) error {
	_, err := c.rndc("reload", zone)

	return err
}

func (c *Client) Freeze(zone string) error {
	_, err := c.rndc("freeze", zone)

	return err
}

func (c *Client) Thaw(zone string) error {
	_, err := c.rndc("thaw", zone)

	return err
}

func (c *Client) FlushAll() error {
	_, err := c.rndc("flush")

	return err
}

func (c *Client) FlushName(name string) error {
	_, err := c.rndc("flushname", name)

	return err
}

func (c *Client) FlushTree(name string) error {
	_, err := c.rndc("flushtree", name)

	return err
}

func (c *Client) DumpStats() error {
	_, err := c.rndc("stats")

	return err
}
