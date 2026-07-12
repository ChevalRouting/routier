package kea

import (
	"encoding/json"
	"sort"
	"strconv"
)

type Stat struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

func (c *Client) Statistics(service string) ([]Stat, error) {
	raw, err := c.send(service, "statistic-get-all", nil)
	if err != nil {
		return nil, err
	}

	var samples map[string][][]json.RawMessage
	if err := json.Unmarshal(raw, &samples); err != nil {
		return nil, err
	}

	out := make([]Stat, 0, len(samples))
	for name, series := range samples {
		if len(series) == 0 || len(series[0]) == 0 {
			continue
		}

		var v float64
		if f, err := strconv.ParseFloat(string(series[0][0]), 64); err == nil {
			v = f
		}

		out = append(out, Stat{Name: name, Value: v})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
