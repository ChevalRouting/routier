package bind

import (
	"fmt"
	"strconv"
	"strings"
)

type Answer struct {
	Name  string `json:"name"`
	TTL   int    `json:"ttl"`
	Class string `json:"class"`
	Type  string `json:"type"`
	Data  string `json:"data"`
}

type QueryResult struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	RCode   string   `json:"rcode"`
	Flags   []string `json:"flags,omitempty"   validate:"optional"`
	Answers []Answer `json:"answers,omitempty" validate:"optional"`
	Server  string   `json:"server,omitempty"  validate:"optional"`
	QueryMS int      `json:"query_ms"`
}

func (c *Client) resolverPort() int {
	if c.ResolverPort > 0 {
		return c.ResolverPort
	}

	return DefaultPort
}

func (c *Client) Query(name, qtype string) (*QueryResult, error) {
	if name == "" {
		return nil, fmt.Errorf("query requires a name")
	}

	if qtype == "" {
		qtype = "A"
	}

	qtype = strings.ToUpper(qtype)
	args := []string{
		"@" + c.resolverAddr(),
		"-p", strconv.Itoa(c.resolverPort()),
		"-t", qtype,
		name,
		"+tries=1",
		"+time=2",
	}

	out, runErr := runner("dig", args...)

	result := parseDig(string(out))
	if result.RCode == "" {
		if runErr != nil {
			return nil, runErr
		}

		return nil, fmt.Errorf("no answer from %s: %s", c.resolverAddr(), strings.TrimSpace(string(out)))
	}

	result.Name = name
	result.Type = qtype

	return result, nil
}

func parseDig(out string) *QueryResult {
	result := &QueryResult{}
	inAnswer := false

	for line := range strings.SplitSeq(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, ";") {
			inAnswer = strings.HasPrefix(trimmed, ";; ANSWER SECTION:")
			setDigMeta(result, trimmed)
			continue
		}

		if !inAnswer {
			continue
		}

		if answer, ok := parseAnswer(trimmed); ok {
			result.Answers = append(result.Answers, answer)
		}
	}

	return result
}

func setDigMeta(result *QueryResult, line string) {
	switch {
	case strings.Contains(line, "->>HEADER<<-"):
		result.RCode = digField(line, "status:", ",")
	case strings.HasPrefix(line, ";; flags:"):
		result.Flags = strings.Fields(digField(line, "flags:", ";"))
	case strings.HasPrefix(line, ";; Query time:"):
		result.QueryMS = firstNumber(line)
	case strings.HasPrefix(line, ";; SERVER:"):
		result.Server = strings.TrimSpace(strings.TrimPrefix(line, ";; SERVER:"))
	}
}

func digField(line, after, until string) string {
	_, tail, ok := strings.Cut(line, after)
	if !ok {
		return ""
	}

	value, _, _ := strings.Cut(tail, until)
	return strings.TrimSpace(value)
}

func firstNumber(line string) int {
	for _, field := range strings.Fields(line) {
		if n, err := strconv.Atoi(field); err == nil {
			return n
		}
	}

	return 0
}

func parseAnswer(line string) (Answer, bool) {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return Answer{}, false
	}

	return Answer{
		Name:  fields[0],
		TTL:   atoi(fields[1]),
		Class: fields[2],
		Type:  fields[3],
		Data:  strings.Join(fields[4:], " "),
	}, true
}
