package failures

import (
	"strings"
	"testing"
)

func TestRedactBytesCoversArtifactFormats(t *testing.T) {
	lines := map[string]string{
		"wireguard":  "PrivateKey = c2VjcmV0",
		"frr":        " neighbor 192.0.2.1 password s3cret",
		"keepalived": "    auth_pass s3cret",
		"bind":       `    secret "c2VjcmV0";`,
		"kea":        `"secret": "c2VjcmV0",`,
		"yaml token": "    token: s3cret",
		"yaml key":   "    key: c2VjcmV0",
		"ospf":       "  auth_key: s3cret",
		"hash":       "    password_hash: $6$s3cret",
	}

	for name, line := range lines {
		out := string(redactBytes([]byte(line)))
		if strings.Contains(out, "s3cret") || strings.Contains(out, "c2VjcmV0") {
			t.Errorf("%s: secret left in %q", name, out)
		}
	}

	if out := string(redactBytes([]byte("public_key: abc"))); out != "public_key: abc" {
		t.Errorf("public key redacted: %q", out)
	}
}
