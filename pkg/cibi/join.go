package cibi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/auth/identity"
	webdb "github.com/ChevalRouting/routier/pkg/db"
)

const (
	KeyName     = "cibi"
	joinTimeout = 90 * time.Second
)

var ErrRefused = errors.New("cibi refused the code")

type Join struct {
	Server   string `yaml:"server"             json:"server"`
	Code     string `yaml:"code"               json:"code"`
	URL      string `yaml:"url,omitempty"      json:"url,omitempty"`
	Port     int    `yaml:"port,omitempty"     json:"port,omitempty"`
	Insecure bool   `yaml:"insecure,omitempty" json:"insecure,omitempty"`
}

type Result struct {
	RoutierID string   `json:"routier_id"`
	Name      string   `json:"name"`
	Networks  []string `json:"networks"`
	Linked    string   `json:"linked"`
	Applying  bool     `json:"applying"`
}

type joinRequest struct {
	Code        string `json:"code"`
	URL         string `json:"url,omitempty"`
	Port        int    `json:"port,omitempty"`
	Token       string `json:"token"`
	Fingerprint string `json:"fingerprint"`
	Hostname    string `json:"hostname"`
}

type joinResponse struct {
	Result *Result `json:"result"`
	Error  string  `json:"error"`
}

func (j Join) Validate() error {
	parsed, err := url.Parse(j.Server)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return fmt.Errorf("cibi address %q must look like https://cibi.example.com", j.Server)
	}

	if strings.TrimSpace(j.Code) == "" {
		return errors.New("the enrollment code is empty")
	}

	return nil
}

func (j Join) Run(ctx context.Context, db *webdb.DB, self *identity.Identity) (Result, error) {
	if err := j.Validate(); err != nil {
		return Result{}, err
	}

	hostname, _ := os.Hostname()
	key, token, err := webdb.NewAPIKey(ctx, db, KeyName+" "+time.Now().UTC().Format("2006-01-02 15:04:05.000"), "cibi join")
	if err != nil {
		return Result{}, fmt.Errorf("create the API key for cibi: %w", err)
	}

	result, err := j.post(ctx, joinRequest{Code: j.Code, URL: j.URL, Port: j.Port, Token: token, Fingerprint: self.Fingerprint(), Hostname: hostname})
	if err != nil {
		if revokeErr := webdb.RevokeAPIKey(context.WithoutCancel(ctx), db, key.ID); revokeErr != nil {
			return result, errors.Join(err, revokeErr)
		}

		return result, err
	}

	return result, revokeOlderKeys(ctx, db, key.ID)
}

func revokeOlderKeys(ctx context.Context, db *webdb.DB, current string) error {
	keys, err := webdb.ListAPIKeys(ctx, db)
	if err != nil {
		return err
	}

	for _, key := range keys {
		if strings.HasPrefix(key.Name, KeyName+" ") && key.ID != current && key.RevokedAt == nil {
			if err := webdb.RevokeAPIKey(ctx, db, key.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (j Join) post(ctx context.Context, req joinRequest) (Result, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return Result{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, joinTimeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(j.Server, "/")+"/api/enroll", bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: j.Insecure} //nolint:gosec
	resp, err := (&http.Client{Transport: transport}).Do(httpReq)
	if err != nil {
		return Result{}, fmt.Errorf("reach cibi: %w", err)
	}

	defer func(action func() error) { _ = action() }(resp.Body.Close)
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Result{}, err
	}

	var decoded joinResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return Result{}, fmt.Errorf("cibi answered %s", resp.Status)
	}

	switch {
	case resp.StatusCode == http.StatusForbidden:
		return Result{}, fmt.Errorf("%w: %s", ErrRefused, decoded.Error)
	case resp.StatusCode != http.StatusOK || decoded.Result == nil:
		return Result{}, fmt.Errorf("cibi answered %s: %s", resp.Status, decoded.Error)
	}

	return *decoded.Result, nil
}

func (r Result) Summary() string {
	parts := []string{"joined cibi as " + r.Name}
	if len(r.Networks) > 0 {
		parts = append(parts, "networks "+strings.Join(r.Networks, ", "))
	}

	if r.Linked != "" {
		parts = append(parts, "linked to "+r.Linked)
	}

	if r.Applying {
		parts = append(parts, "cibi is applying the configuration")
	}

	return strings.Join(parts, "; ")
}
