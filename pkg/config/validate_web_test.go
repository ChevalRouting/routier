package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"
)

var testCAPEM = selfSignedPEM()

func selfSignedPEM() string {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test ca"}, NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func webErrors(tls *WebTLS) []string {
	var messages []string
	add := func(msg string, args ...any) { messages = append(messages, fmt.Sprintf(msg, args...)) }
	cfg := &Config{Interfaces: map[string]*Interface{"wan": {Type: "ethernet"}}, Web: &Web{TLS: tls}}
	validateWeb(cfg, add)
	return messages
}

func TestValidateWebAcceptsGoodDefinitions(t *testing.T) {
	good := []*WebTLS{
		{},
		{CertFile: "custom.crt", KeyFile: "custom.key"},
		{ACME: &WebACME{Domains: []string{"rt1.example.net"}}},
		{ACME: &WebACME{Directory: "https://ca.internal/acme/directory", Domains: []string{"rt1.internal.example"}, CAPEM: testCAPEM, EABKID: "kid", EABKey: "c2VjcmV0", HTTPPort: 8081}},
	}
	for _, tls := range good {
		if errs := webErrors(tls); len(errs) != 0 {
			t.Fatalf("unexpected errors for %+v: %v", tls, errs)
		}
	}
}

func TestValidateWebReportsMistakes(t *testing.T) {
	cases := map[string]*WebTLS{
		"cert_file and key_file must be set together":           {CertFile: "custom.crt"},
		`web.tls.cert_file "../etc/shadow" must be a file name`: {CertFile: "../etc/shadow", KeyFile: "custom.key"},
		"set either cert_file and key_file or acme, not both":   {CertFile: "a.crt", KeyFile: "a.key", ACME: &WebACME{Domains: []string{"rt1.example.net"}}},
		"web.tls.acme.domains is required":                      {ACME: &WebACME{}},
		`"localhost" must be a fully qualified domain name`:     {ACME: &WebACME{Domains: []string{"localhost"}}},
		`"rt1.example.net" is duplicated`:                       {ACME: &WebACME{Domains: []string{"rt1.example.net", "RT1.example.net."}}},
		"web.tls.acme.directory must be an https URL":           {ACME: &WebACME{Directory: "http://ca", Domains: []string{"rt1.example.net"}}},
		"web.tls.acme.http_port must be between 1 and 65535":    {ACME: &WebACME{HTTPPort: 70000, Domains: []string{"rt1.example.net"}}},
		"eab_kid and eab_key must be set together":              {ACME: &WebACME{EABKID: "kid", Domains: []string{"rt1.example.net"}}},
		"web.tls.acme.eab_key must be base64url encoded":        {ACME: &WebACME{EABKID: "kid", EABKey: "not base64!", Domains: []string{"rt1.example.net"}}},
		`web.tls.acme.ca_pem must contain a PEM certificate`:    {ACME: &WebACME{CAPEM: "not a certificate", Domains: []string{"rt1.example.net"}}},
	}
	for want, tls := range cases {
		errs := strings.Join(webErrors(tls), "\n")
		if !strings.Contains(errs, want) {
			t.Errorf("want %q, got %q", want, errs)
		}
	}
}
