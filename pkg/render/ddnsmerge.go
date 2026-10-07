package render

import (
	"strconv"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
)

func DDNSManagedZone(cfg *config.Config, name string) bool {
	return ddnsManagedZone(cfg, name)
}

func MergeDDNSZone(cfg *config.Config, zoneName, frozen string) string {
	z := ddnsDeclaredZone(cfg, zoneName)

	records, serial := parseZoneFile(frozen, config.NormalizeDNSName(zoneName))

	dynamicOwner := map[string]bool{}
	for _, r := range records {
		if r.rtype == "DHCID" {
			dynamicOwner[r.owner] = true
		}
	}

	staticOwner := map[string]bool{}
	for _, l := range zoneLines(z) {
		staticOwner[l.owner] = true
	}

	var preserved []zoneLine
	for _, r := range records {
		if !dynamicOwner[r.owner] || staticOwner[r.owner] {
			continue
		}

		if r.rtype == "SOA" || r.rtype == "NS" {
			continue
		}

		preserved = append(preserved, r)
	}

	return renderZoneFileWith(z, strconv.Itoa(serial+1), preserved)
}

func ddnsDeclaredZone(cfg *config.Config, name string) config.DNSZone {
	if cfg != nil && cfg.DNS != nil && cfg.DNS.Server != nil {
		for _, z := range cfg.DNS.Server.Zones {
			if strings.EqualFold(config.NormalizeDNSName(z.Name), config.NormalizeDNSName(name)) {
				return z
			}
		}
	}

	return ddnsBootstrapZone(cfg, name)
}

var zoneRecordClasses = map[string]bool{"IN": true, "CH": true, "HS": true, "CS": true}

func parseZoneFile(content, origin string) ([]zoneLine, int) {
	var (
		lines     []zoneLine
		serial    int
		curOrigin = origin
		lastOwner string
	)

	for _, raw := range logicalLines(content) {
		trimmed := strings.TrimRight(raw.text, " \t")
		if trimmed == "" {
			continue
		}

		fields := strings.Fields(trimmed)
		switch strings.ToUpper(fields[0]) {
		case "$ORIGIN":
			if len(fields) > 1 {
				curOrigin = config.NormalizeDNSName(fields[1])
			}

			continue
		case "$TTL", "$INCLUDE", "$GENERATE":
			continue
		}

		owner := lastOwner
		if !raw.ownerBlank {
			owner = fields[0]
			fields = fields[1:]
		}

		if len(fields) == 0 {
			continue
		}

		lastOwner = owner
		fields = trimTTLAndClass(fields)
		if len(fields) < 2 {
			continue
		}

		rtype := strings.ToUpper(fields[0])
		data := strings.Join(fields[1:], " ")

		if rtype == "SOA" {
			if soaFields := strings.Fields(data); len(soaFields) >= 3 {
				if n, err := strconv.Atoi(soaFields[2]); err == nil {
					serial = n
				}
			}
		}

		lines = append(lines, zoneLine{
			owner: relativeOwner(makeAbsolute(owner, curOrigin), origin),
			rtype: rtype,
			data:  data,
		})
	}

	return lines, serial
}

func trimTTLAndClass(fields []string) []string {
	for len(fields) > 0 {
		if zoneRecordClasses[strings.ToUpper(fields[0])] {
			fields = fields[1:]
			continue
		}

		if _, err := strconv.Atoi(fields[0]); err == nil {
			fields = fields[1:]
			continue
		}

		break
	}

	return fields
}

type logicalLine struct {
	text       string
	ownerBlank bool
}

func logicalLines(content string) []logicalLine {
	var (
		out   []logicalLine
		buf   strings.Builder
		depth int
		blank bool
	)

	flush := func() { logicalLinesCallback(&out, &buf, blank) }

	for _, line := range strings.Split(content, "\n") {
		stripped := stripComment(line)
		if depth == 0 {
			flush()
			blank = strings.TrimSpace(line) != "" && (line[0] == ' ' || line[0] == '\t')
		}

		for _, r := range stripped {
			switch r {
			case '(':
				depth++
			case ')':
				if depth > 0 {
					depth--
				}
			}
		}

		if buf.Len() > 0 {
			_ = buf.WriteByte(' ')
		}

		_, _ = buf.WriteString(strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(stripped, "(", " "), ")", " ")))

		if depth == 0 {
			flush()
		}
	}

	flush()

	return out
}

func stripComment(line string) string {
	var b strings.Builder
	inQuote := false
	for _, r := range line {
		switch r {
		case '"':
			inQuote = !inQuote
		case ';':
			if !inQuote {
				return b.String()
			}
		}

		_, _ = b.WriteRune(r)
	}

	return b.String()
}

func makeAbsolute(name, origin string) string {
	if name == "@" {
		return strings.ToLower(origin)
	}

	name = strings.ToLower(name)
	if strings.HasSuffix(name, ".") {
		return name
	}

	if origin == "." || origin == "" {
		return name + "."
	}

	return name + "." + strings.ToLower(origin)
}

func relativeOwner(absolute, origin string) string {
	absolute = strings.ToLower(absolute)
	origin = strings.ToLower(origin)
	if absolute == origin {
		return "@"
	}

	if suffix := "." + origin; strings.HasSuffix(absolute, suffix) {
		return strings.TrimSuffix(absolute, suffix)
	}

	return strings.TrimSuffix(absolute, ".")
}

func logicalLinesCallback(out *[]logicalLine, buf *strings.Builder, blank bool) {
	if (*buf).Len() > 0 {
		(*out) = append((*out), logicalLine{text: (*buf).String(), ownerBlank: blank})
		(*buf).Reset()
	}
}
