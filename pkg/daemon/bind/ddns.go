package bind

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultTSIGAlgorithm = "hmac-sha256"

var ddnsHiddenTypes = map[string]bool{
	"SOA": true, "NS": true, "DHCID": true,
	"RRSIG": true, "NSEC": true, "NSEC3": true, "NSEC3PARAM": true,
	"DNSKEY": true, "CDS": true, "CDNSKEY": true, "TYPE65534": true,
}

func (c *Client) DynamicRecords(zone string) ([]ZoneRecord, error) {
	keyFile, cleanup, err := c.writeTSIGKeyFile()
	if err != nil {
		return nil, err
	}

	defer cleanup()

	args := []string{
		"@" + c.resolverAddr(),
		"-p", strconv.Itoa(c.resolverPort()),
		"-k", keyFile,
		"-t", "AXFR",
		zone,
		"+tries=1",
		"+time=2",
	}

	out, runErr := runner("dig", args...)
	text := string(out)

	if strings.Contains(text, "Transfer failed") || strings.Contains(text, "communications error") || strings.Contains(text, "connection refused") {
		return nil, fmt.Errorf("axfr %s: %s", zone, firstLine(strings.TrimSpace(text)))
	}

	records := parseAXFR(text)
	if records == nil && runErr != nil {
		return nil, fmt.Errorf("axfr %s: %w: %s", zone, runErr, strings.TrimSpace(text))
	}

	return records, nil
}

func (c *Client) DeleteRecord(zone, name, rtype, value string) error {
	keyFile, cleanup, err := c.writeTSIGKeyFile()
	if err != nil {
		return err
	}

	defer cleanup()

	script, cleanupScript, err := writeTempFile("routier-nsupdate-", c.deleteScript(zone, name, rtype, value))
	if err != nil {
		return err
	}

	defer cleanupScript()

	out, err := runner("nsupdate", "-k", keyFile, script)
	if err != nil {
		return fmt.Errorf("nsupdate delete %s %s: %w: %s", name, rtype, err, strings.TrimSpace(string(out)))
	}

	return nil
}

func (c *Client) deleteScript(zone, name, rtype, value string) string {
	rtype = strings.ToUpper(strings.TrimSpace(rtype))

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "server %s %d\n", c.resolverAddr(), c.resolverPort())
	_, _ = fmt.Fprintf(&b, "zone %s\n", zone)

	if value = strings.TrimSpace(value); value != "" {
		_, _ = fmt.Fprintf(&b, "update delete %s %s %s\n", name, rtype, value)
	} else {
		_, _ = fmt.Fprintf(&b, "update delete %s %s\n", name, rtype)
	}

	if rtype == "A" || rtype == "AAAA" {
		_, _ = fmt.Fprintf(&b, "update delete %s DHCID\n", name)
	}

	_, _ = b.WriteString("send\n")

	return b.String()
}

func (c *Client) writeTSIGKeyFile() (string, func(), error) {
	if c.TSIGSecret == "" {
		return "", nil, fmt.Errorf("ddns requires a TSIG key")
	}

	algo := c.TSIGAlgorithm
	if algo == "" {
		algo = defaultTSIGAlgorithm
	}

	content := fmt.Sprintf("key \"%s\" {\n\talgorithm %s;\n\tsecret \"%s\";\n};\n", c.TSIGName, algo, c.TSIGSecret)

	return writeTempFile("routier-tsig-", content)
}

func writeTempFile(prefix, content string) (string, func(), error) {
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}

	cleanup := func() { _ = os.Remove(f.Name()) }

	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, err
	}

	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, err
	}

	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, err
	}

	return f.Name(), cleanup, nil
}

func parseAXFR(out string) []ZoneRecord {
	var records []ZoneRecord
	for line := range strings.SplitSeq(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}

		answer, ok := parseAnswer(trimmed)
		if !ok || ddnsHiddenTypes[strings.ToUpper(answer.Type)] {
			continue
		}

		records = append(records, ZoneRecord{
			Name:  answer.Name,
			Type:  strings.ToUpper(answer.Type),
			Value: answer.Data,
			TTL:   answer.TTL,
		})
	}

	return records
}
