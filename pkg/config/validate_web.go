package config

import (
	"crypto/x509"
	"encoding/base64"
	"regexp"
	"strings"
)

var webTLSFilePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

var webTLSDomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

func (a *WebACME) NormalizedDomains() []string {
	domains := make([]string, 0, len(a.Domains))
	for _, domain := range a.Domains {
		domains = append(domains, NormalizeDomain(domain))
	}

	return domains
}

func ValidWebTLSFileName(name string) bool {
	return webTLSFilePattern.MatchString(name)
}

func NormalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
}

func validateWeb(cfg *Config, add addfunc) {
	if cfg.Web == nil || cfg.Web.TLS == nil {
		return
	}

	t := cfg.Web.TLS
	if (t.CertFile == "") != (t.KeyFile == "") {
		add("web.tls: cert_file and key_file must be set together")
	}

	for field, name := range map[string]string{"cert_file": t.CertFile, "key_file": t.KeyFile} {
		if name != "" && !webTLSFilePattern.MatchString(name) {
			add("web.tls.%s %q must be a file name in /var/lib/routier/tls, without a directory", field, name)
		}
	}

	if t.ACME != nil && t.CertFile != "" {
		add("web.tls: set either cert_file and key_file or acme, not both")
	}

	if t.ACME != nil {
		validateWebACME(cfg, t.ACME, add)
	}
}

func validateWebACME(cfg *Config, a *WebACME, add addfunc) {
	if len(a.Domains) == 0 {
		add("web.tls.acme.domains is required")
	}

	seen := map[string]bool{}
	for _, domain := range a.NormalizedDomains() {
		if !webTLSDomainPattern.MatchString(domain) {
			add("web.tls.acme.domains: %q must be a fully qualified domain name", domain)
		}

		if seen[domain] {
			add("web.tls.acme.domains: %q is duplicated", domain)
		}

		seen[domain] = true
	}

	if a.Directory != "" && !strings.HasPrefix(a.Directory, "https://") {
		add("web.tls.acme.directory must be an https URL")
	}

	if a.HTTPPort != 0 && !isValidPort(a.HTTPPort) {
		add("web.tls.acme.http_port must be between 1 and 65535")
	}

	if a.CAPEM != "" && !x509.NewCertPool().AppendCertsFromPEM([]byte(a.CAPEM)) {
		add("web.tls.acme.ca_pem must contain a PEM certificate")
	}

	if (a.EABKID == "") != (a.EABKey == "") {
		add("web.tls.acme: eab_kid and eab_key must be set together")
	}

	if _, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(a.EABKey, "=")); a.EABKey != "" && err != nil {
		add("web.tls.acme.eab_key must be base64url encoded")
	}
}
