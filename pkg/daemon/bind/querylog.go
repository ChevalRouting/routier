package bind

import (
	"regexp"
	"strings"
)

type LogQuery struct {
	Time   string `json:"time"`
	Client string `json:"client"`
	Port   string `json:"port"`
	Name   string `json:"name"`
	Class  string `json:"class"`
	Type   string `json:"type"`
	Flags  string `json:"flags"`
	Dest   string `json:"dest,omitempty"`
	Raw    string `json:"raw"`
}

var (
	queryTimeRe   = regexp.MustCompile(`^(\d{2}-[A-Za-z]{3}-\d{4} \d{2}:\d{2}:\d{2}\.\d{3})`)
	queryClientRe = regexp.MustCompile(`client (?:@0x[0-9a-fA-F]+ )?([0-9A-Fa-f:.]+)#(\d+)(?: \([^)]*\))?: query: (\S+) (\S+) (\S+) (\S+)(?: \(([0-9A-Fa-f:.]+)\))?`)
)

func ParseQueryLog(line string) (LogQuery, bool) {
	m := queryClientRe.FindStringSubmatch(line)
	if m == nil {
		return LogQuery{}, false
	}

	q := LogQuery{
		Client: m[1],
		Port:   m[2],
		Name:   strings.TrimSuffix(m[3], "."),
		Class:  m[4],
		Type:   m[5],
		Flags:  m[6],
		Dest:   m[7],
		Raw:    line,
	}

	if q.Name == "" {
		q.Name = "."
	}

	if t := queryTimeRe.FindStringSubmatch(line); t != nil {
		q.Time = t[1]
	}

	return q, true
}
