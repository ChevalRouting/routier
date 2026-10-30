package live

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

const passwordAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func Password(length int) (string, error) {
	var b strings.Builder
	limit := big.NewInt(int64(len(passwordAlphabet)))
	for range length {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}

		_ = b.WriteByte(passwordAlphabet[n.Int64()])
	}

	return b.String(), nil
}

func Banner(header string, urls []string, password string) string {
	var b strings.Builder
	_, _ = b.WriteString(header)
	_, _ = b.WriteString("  This system runs from the installation media.\n\n")
	_, _ = b.WriteString("  Install from a browser:\n")
	if len(urls) == 0 {
		_, _ = b.WriteString("    no address yet, find it with 'ip -4 addr' once the network is up\n")
	}

	for _, url := range urls {
		_, _ = fmt.Fprintf(&b, "    %s\n", url)
	}

	if password != "" {
		_, _ = fmt.Fprintf(&b, "  Sign in as routier with the password %s\n\n", password)
	} else {
		_, _ = b.WriteString("  Sign in as routier\n\n")
	}

	_, _ = b.WriteString("  Or install from this console: setup-routier\n\n")
	return b.String()
}

func EnsureCert(certPath, keyPath, host string) (bool, error) {
	if _, err := os.Stat(keyPath); err == nil {
		return false, nil
	}

	return true, writeCert(certPath, keyPath, host)
}

func writeCert(certPath, keyPath, host string) error {

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	if err != nil {
		return err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host},
		DNSNames:     []string{host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}

	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		return err
	}

	return os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644)
}
