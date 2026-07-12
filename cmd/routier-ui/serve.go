package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ChevalRouting/routier/pkg/api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const defaultSecretFile = "/var/lib/routier/jwt.secret"
const defaultTLSCert = "/var/lib/routier/tls.crt"
const defaultTLSKey = "/var/lib/routier/tls.key"

func serveCmd() *cobra.Command {
	var (
		configPath string
		dbPath     string
		addr       string
		secretFile string
		tlsCert    string
		tlsKey     string
		devDir     string
		debug      bool
		logLevel   string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "start the web UI server",
		RunE: func(_ *cobra.Command, _ []string) error {
			lvl, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				return fmt.Errorf("invalid log level %q", logLevel)
			}

			zerolog.SetGlobalLevel(lvl)
			jwtSecret, err := loadOrCreateSecret(secretFile)
			if err != nil {
				return fmt.Errorf("jwt secret: %w", err)
			}

			srv, err := api.New(configPath, dbPath, jwtSecret, debug)
			if err != nil {
				return fmt.Errorf("create server: %w", err)
			}

			var staticFS fs.FS
			if devDir != "" {
				log.Info().Str("dir", devDir).Msg("serving static files from directory")
				staticFS = os.DirFS(devDir)
			} else {
				entries, err := api.EmbeddedFS.ReadDir(".")
				log.Debug().Bool("exist", len(entries) > 0).Any("files", entries).Msg("embed exist")
				if err == nil && len(entries) > 0 {
					sub, err := fs.Sub(api.EmbeddedFS, "dist")
					if err == nil {
						staticFS = sub
						log.Info().Msg("serving embedded static files")
					}
				}
			}

			if staticFS != nil {
				srv.SetStaticFS(staticFS)
			}

			handler := srv.Handler()

			_, certErr := os.Stat(tlsCert)
			_, keyErr := os.Stat(tlsKey)
			tlsEnabled := certErr == nil && keyErr == nil
			if _, portStr, aerr := net.SplitHostPort(addr); aerr == nil {
				if port, perr := strconv.Atoi(portStr); perr == nil {
					srv.SetAdvertise(port, tlsEnabled)
				}
			}

			if tlsEnabled {
				log.Info().Str("addr", addr).Str("cert", tlsCert).Str("key", tlsKey).Msg("starting web server (TLS)")
				return http.ListenAndServeTLS(addr, tlsCert, tlsKey, handler)
			}

			log.Info().Str("addr", addr).Str("config", configPath).Msg("starting web server")
			return http.ListenAndServe(addr, handler)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "/etc/routier/config.yml", "path to config YAML")
	cmd.Flags().StringVar(&dbPath, "db", "/var/lib/routier/web.db", "path to SQLite database")
	cmd.Flags().StringVar(&addr, "addr", ":8080", "listen address")
	cmd.Flags().StringVar(&secretFile, "secret-file", defaultSecretFile, "path to JWT signing secret file (created if absent)")
	cmd.Flags().StringVar(&tlsCert, "tls-cert", defaultTLSCert, "TLS certificate (TLS enabled when both cert and key exist)")
	cmd.Flags().StringVar(&tlsKey, "tls-key", defaultTLSKey, "TLS private key")
	cmd.Flags().StringVar(&devDir, "dev-dir", "", "serve static files from this directory instead of embedded FS")
	cmd.Flags().BoolVar(&debug, "debug", false, "include error details and stack traces in API responses")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	return cmd
}

func loadOrCreateSecret(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		log.Info().Str("file", path).Msg("loaded JWT secret")
		return data, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}

	secret := []byte(hex.EncodeToString(b))

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return nil, fmt.Errorf("create secret dir: %w", err)
	}

	if err := os.WriteFile(path, secret, 0600); err != nil {
		return nil, fmt.Errorf("write secret file: %w", err)
	}

	log.Info().Str("file", path).Msg("generated new JWT secret")
	return secret, nil
}
