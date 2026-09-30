package bondstat

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const procBonding = "/proc/net/bonding"

type Slave struct {
	Name         string `json:"name"`
	MIIStatus    string `json:"mii_status"`
	Speed        int    `json:"speed,omitempty" validate:"optional"`
	Duplex       string `json:"duplex,omitempty" validate:"optional"`
	LinkFailures int    `json:"link_failures"`
	AggregatorID int    `json:"aggregator_id,omitempty" validate:"optional"`
	ActorChurn   string `json:"actor_churn,omitempty" validate:"optional"`
	PartnerChurn string `json:"partner_churn,omitempty" validate:"optional"`
}

type Status struct {
	Name             string  `json:"name"`
	Mode             string  `json:"mode"`
	MIIStatus        string  `json:"mii_status"`
	LACP             bool    `json:"lacp"`
	LACPRate         string  `json:"lacp_rate,omitempty" validate:"optional"`
	XmitHashPolicy   string  `json:"xmit_hash_policy,omitempty" validate:"optional"`
	ActiveAggregator int     `json:"active_aggregator,omitempty" validate:"optional"`
	Healthy          bool    `json:"healthy"`
	Slaves           []Slave `json:"slaves,omitempty" validate:"optional"`
}

func List() []Status {
	entries, err := os.ReadDir(procBonding)
	if err != nil {
		return nil
	}

	var out []Status
	for _, e := range entries {
		if s := Read(e.Name()); s != nil {
			out = append(out, *s)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func Read(name string) *Status {
	data, err := os.ReadFile(filepath.Join(procBonding, name))
	if err != nil {
		return nil
	}

	return Parse(name, string(data))
}

func Parse(name, content string) *Status {
	s := &Status{Name: name}
	var slave *Slave

	for _, line := range strings.Split(content, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if key == "Slave Interface" {
			s.Slaves = append(s.Slaves, Slave{Name: value})
			slave = &s.Slaves[len(s.Slaves)-1]
			continue
		}

		if slave == nil {
			applyHeader(s, key, value)
			continue
		}

		applySlave(slave, key, value)
	}

	s.Healthy = healthy(s)
	return s
}

func applyHeader(s *Status, key, value string) {
	switch key {
	case "Bonding Mode":
		s.Mode = value
		s.LACP = strings.Contains(value, "802.3ad")
	case "Transmit Hash Policy":
		s.XmitHashPolicy = trimCode(value)
	case "MII Status":
		s.MIIStatus = value
	case "LACP rate", "LACP active":
		if key == "LACP rate" {
			s.LACPRate = value
		}
	case "Aggregator ID":
		if s.ActiveAggregator == 0 {
			s.ActiveAggregator = atoi(value)
		}
	}
}

func applySlave(slave *Slave, key, value string) {
	switch key {
	case "MII Status":
		slave.MIIStatus = value
	case "Speed":
		slave.Speed = atoi(strings.TrimSuffix(value, " Mbps"))
	case "Duplex":
		slave.Duplex = value
	case "Link Failure Count":
		slave.LinkFailures = atoi(value)
	case "Aggregator ID":
		slave.AggregatorID = atoi(value)
	case "Actor Churn State":
		slave.ActorChurn = value
	case "Partner Churn State":
		slave.PartnerChurn = value
	}
}

func healthy(s *Status) bool {
	if s.MIIStatus != "up" || len(s.Slaves) == 0 {
		return false
	}

	for _, sl := range s.Slaves {
		if sl.MIIStatus != "up" {
			return false
		}

		if s.LACP && (s.ActiveAggregator == 0 || sl.AggregatorID != s.ActiveAggregator) {
			return false
		}

		if s.LACP && (churned(sl.ActorChurn) || churned(sl.PartnerChurn)) {
			return false
		}
	}

	return true
}

func churned(state string) bool {
	state = strings.ToLower(strings.TrimSpace(state))
	return state != "" && state != "none"
}

func trimCode(value string) string {
	if i := strings.Index(value, " ("); i >= 0 {
		return value[:i]
	}

	return value
}

func atoi(value string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(value))
	return n
}
