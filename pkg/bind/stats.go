package bind

import (
	"os"
	"strings"
)

const StatsFile = "/var/bind/named.stats"

type Stat struct {
	Name    string  `json:"name"`
	Section string  `json:"section,omitempty" validate:"optional"`
	Value   float64 `json:"value"`
}

var KeyStats = []string{
	"QUERY",
	"IPv4 requests received",
	"IPv6 requests received",
	"responses sent",
	"queries resulted in successful answer",
	"queries resulted in authoritative answer",
	"queries resulted in non authoritative answer",
	"queries resulted in NXDOMAIN",
	"queries resulted in SERVFAIL",
	"queries resulted in nxrrset",
	"queries caused recursion",
	"requests with EDNS(0) received",
	"recursive queries rejected",
	"queries dropped",
	"cache hits",
	"cache misses",
}

func (c *Client) Statistics() ([]Stat, error) {
	if err := c.DumpStats(); err != nil {
		return nil, err
	}

	return ReadStats(StatsFile)
}

func ReadStats(path string) ([]Stat, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseStats(string(raw)), nil
}

func ParseStats(dump string) []Stat {
	var out []Stat

	section := ""
	seen := map[string]bool{}

	for _, raw := range strings.Split(dump, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "++") {
			section = strings.TrimSpace(strings.Trim(line, "+ "))
			continue
		}

		if strings.HasPrefix(line, "[") || strings.HasPrefix(line, "---") {
			continue
		}

		count, name, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}

		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		key := section + "/" + name
		if seen[key] {
			continue
		}

		seen[key] = true
		out = append(out, Stat{Name: name, Section: section, Value: atof(count)})
	}

	return out
}

func StatValue(stats []Stat, name string) float64 {
	for _, s := range stats {
		if s.Name == name {
			return s.Value
		}
	}

	return 0
}

func KeyStatistics(stats []Stat) []Stat {
	byName := make(map[string]Stat, len(stats))
	for _, s := range stats {
		if _, ok := byName[s.Name]; !ok {
			byName[s.Name] = s
		}
	}

	out := make([]Stat, 0, len(KeyStats))
	for _, name := range KeyStats {
		if s, ok := byName[name]; ok {
			out = append(out, s)
		}
	}

	return out
}
