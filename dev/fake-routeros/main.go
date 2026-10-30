package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"flag"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type apiError struct {
	Error   int    `json:"error"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

type server struct {
	store    *store
	username string
	password string
}

func main() {
	addr := flag.String("addr", ":443", "listen address")
	data := flag.String("data", "crs310.json", "initial device state")
	username := flag.String("username", "routier", "API user")
	password := flag.String("password", "routier", "API password")
	flag.Parse()

	s, err := loadStore(*data)
	if err != nil {
		log.Fatal().Err(err).Msg("load device state")
	}

	certificate, err := selfSigned()
	if err != nil {
		log.Fatal().Err(err).Msg("create certificate")
	}

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           &server{store: s, username: *username, password: *password},
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{certificate}},
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Info().Str("addr", *addr).Msg("fake RouterOS REST API listening")
	log.Fatal().Err(httpServer.ListenAndServeTLS("", "")).Msg("serve")
}

func selfSigned() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "fake-routeros"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, password, ok := r.BasicAuth()
	if !ok || user != s.username || password != s.password {
		reply(w, http.StatusUnauthorized, apiError{Error: 401, Message: "Unauthorized"})
		return
	}

	path, ok := strings.CutPrefix(r.URL.Path, "/rest")
	if !ok {
		reply(w, http.StatusNotFound, apiError{Error: 404, Message: "Not Found"})
		return
	}

	log.Info().Str("method", r.Method).Str("path", path).Msg("request")
	var body record
	if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodDelete {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	s.handle(w, r.Method, path, body)
}

func (s *server) handle(w http.ResponseWriter, method, path string, body record) {
	var err error
	switch method {
	case http.MethodGet:
		value, ok := s.store.get(path)
		if !ok {
			reply(w, http.StatusNotFound, apiError{Error: 404, Message: "Not Found"})
			return
		}

		reply(w, http.StatusOK, value)
		return
	case http.MethodPost:
		reply(w, http.StatusOK, []record{{"name": "ether1", "status": "link-ok", "rate": "2.5Gbps"}})
		return
	case http.MethodPut:
		var created record
		created, err = s.store.put(path, body)
		if err == nil {
			reply(w, http.StatusCreated, created)
			return
		}
	case http.MethodPatch:
		err = s.store.patch(path, body)
	case http.MethodDelete:
		err = s.store.remove(path)
	}

	if err != nil {
		status := http.StatusNotFound
		if errors.Is(err, errConflict) {
			status = http.StatusBadRequest
		}

		reply(w, status, apiError{Error: status, Message: http.StatusText(status), Detail: err.Error()})
		return
	}

	reply(w, http.StatusOK, record{})
}
