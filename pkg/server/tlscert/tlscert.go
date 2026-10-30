package tlscert

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ChevalRouting/certs"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

const DefaultDir = "/var/lib/routier"

const watchInterval = 5 * time.Second

var (
	ErrInvalidFile = errors.New("invalid tls file")
	ErrNotACME     = certs.ErrNotACME
	ErrRenewing    = certs.ErrRenewing
)

type Paths struct {
	Dir      string
	CertFile string
	KeyFile  string
}

type Provider struct {
	paths    Paths
	certs    *certs.Provider
	settings config.WebTLS
}

type Status struct {
	Mode         string        `json:"mode"`
	Error        string        `json:"error,omitempty" validate:"optional"`
	Certificates []Certificate `json:"certificates"`
	Files        []string      `json:"files"`
	Log          []LogEntry    `json:"log"`
}

type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Domain  string    `json:"domain,omitempty" validate:"optional"`
	Message string    `json:"message"`
}

type Certificate struct {
	Name     string    `json:"name"`
	Source   string    `json:"source"`
	Issuer   string    `json:"issuer,omitempty" validate:"optional"`
	NotAfter time.Time `json:"not_after,omitempty" validate:"optional"`
	Ready    bool      `json:"ready"`
	Error    string    `json:"error,omitempty" validate:"optional"`
}

func DefaultPaths() Paths {
	return Paths{Dir: DefaultDir, CertFile: filepath.Join(DefaultDir, "tls.crt"), KeyFile: filepath.Join(DefaultDir, "tls.key")}
}

func (p Paths) filesDir() string { return filepath.Join(p.Dir, "tls") }

func (p Paths) File(name string) string { return filepath.Join(p.filesDir(), name) }

func (p Paths) acmeCAFile() string { return filepath.Join(p.Dir, "acme-ca.pem") }

func (p Paths) settings(web config.WebTLS) certs.Settings {
	settings := certs.Settings{}
	if web.CertFile != "" {
		settings.CertFile, settings.KeyFile = p.File(web.CertFile), p.File(web.KeyFile)
	}

	if a := web.ACME; a != nil {
		settings.ACME = &certs.ACME{Directory: a.Directory, Email: a.Email, Domains: a.Domains, EABKID: a.EABKID, EABKey: a.EABKey}
		if a.HTTPPort != 0 {
			settings.ACME.ChallengeAddr = ":" + strconv.Itoa(a.HTTPPort)
		}

		if a.CAPEM != "" {
			settings.ACME.CAFile = p.acmeCAFile()
		}
	}

	return settings
}

func (p Paths) writeACMECA(web config.WebTLS) error {
	if web.ACME == nil || web.ACME.CAPEM == "" {
		return nil
	}

	path := p.acmeCAFile()
	if current, err := os.ReadFile(path); err == nil && string(current) == web.ACME.CAPEM {
		return nil
	}

	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(web.ACME.CAPEM), 0o644)
}

func New(paths Paths, settings config.WebTLS) (*Provider, error) {
	if err := paths.writeACMECA(settings); err != nil {
		return nil, err
	}

	provider, err := certs.New(certs.Options{Dir: paths.Dir, FallbackCert: paths.CertFile, FallbackKey: paths.KeyFile}, paths.settings(settings))
	if err != nil {
		return nil, err
	}

	return &Provider{paths: paths, certs: provider, settings: settings}, nil
}

func (p *Provider) TLSConfig() *tls.Config { return p.certs.TLSConfig() }

func (p *Provider) Watch(ctx context.Context, path string) {
	p.certs.Start(ctx)
	var modified time.Time
	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()
	for {
		if info, err := os.Stat(path); err == nil && !info.ModTime().Equal(modified) {
			modified = info.ModTime()
			p.reload(path)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *Provider) reload(path string) {
	settings, err := LoadSettings(path)
	if err != nil {
		log.Warn().Err(err).Str("config", path).Msg("read web tls settings")
		return
	}

	if reflect.DeepEqual(settings, p.settings) {
		return
	}

	if err := p.paths.writeACMECA(settings); err != nil {
		log.Error().Err(err).Msg("write acme ca bundle; keeping the current certificate")
		return
	}

	if err := p.certs.Apply(p.paths.settings(settings)); err != nil {
		log.Error().Err(err).Msg("apply web tls settings; keeping the current certificate")
		return
	}

	p.settings = settings
	log.Info().Str("mode", p.certs.Settings().Mode()).Msg("web tls settings applied")
}

func LoadSettings(path string) (config.WebTLS, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return config.WebTLS{}, err
	}

	if cfg.Web == nil || cfg.Web.TLS == nil {
		return config.WebTLS{}, nil
	}

	return *cfg.Web.TLS, nil
}

func (p *Provider) Status() Status {
	status := p.certs.Status()
	converted := Status{Mode: status.Mode, Error: status.Error, Certificates: []Certificate{}, Files: p.files(), Log: []LogEntry{}}
	for _, c := range status.Certificates {
		converted.Certificates = append(converted.Certificates, Certificate{Name: c.Name, Source: c.Source, Issuer: c.Issuer, NotAfter: c.NotAfter, Ready: c.Ready, Error: c.Error})
	}

	for _, entry := range status.Log {
		converted.Log = append(converted.Log, LogEntry{Time: entry.Time, Level: entry.Level, Domain: entry.Domain, Message: entry.Message})
	}

	return converted
}

func (p *Provider) Renew(domain string) error { return p.certs.Renew(domain) }

func (p *Provider) files() []string {
	names := []string{}
	entries, _ := os.ReadDir(p.paths.filesDir())
	for _, entry := range entries {
		if entry.Type().IsRegular() && config.ValidWebTLSFileName(entry.Name()) {
			names = append(names, entry.Name())
		}
	}

	return names
}

func (p *Provider) ReadCertificate(name string) (string, error) {
	if !config.ValidWebTLSFileName(name) {
		return "", fmt.Errorf("%w: %q must be a file name without a directory", ErrInvalidFile, name)
	}

	data, err := os.ReadFile(p.paths.File(name))
	if err != nil {
		return "", err
	}

	if strings.Contains(string(data), "PRIVATE KEY") {
		return "", fmt.Errorf("%w: %q holds a private key", ErrInvalidFile, name)
	}

	return string(data), nil
}

func (p *Provider) StoreFile(name string, data []byte) error {
	if !config.ValidWebTLSFileName(name) {
		return fmt.Errorf("%w: %q must be a file name without a directory", ErrInvalidFile, name)
	}

	err := certs.WriteFile(p.paths.File(name), data)
	if errors.Is(err, certs.ErrInvalidPEM) {
		return fmt.Errorf("%w: %w", ErrInvalidFile, err)
	}

	return err
}
