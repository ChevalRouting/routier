package routeros

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	requestTimeout   = 15 * time.Second
	maxResponseBytes = 8 << 20
)

type Client struct {
	base     string
	username string
	password string
	http     *http.Client
}

type ClientConfig struct {
	Host          string
	Scheme        string
	Port          string
	Username      string
	Password      string
	TLSSkipVerify bool
}

type apiError struct {
	Error   int    `json:"error"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func NewClient(config ClientConfig) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: config.TLSSkipVerify}

	host := config.Host
	if _, _, err := net.SplitHostPort(host); err != nil && config.Port != "" {
		host = net.JoinHostPort(strings.Trim(host, "[]"), config.Port)
	}

	return &Client{
		base:     schemeOrDefault(config.Scheme) + "://" + host + "/rest",
		username: config.Username,
		password: config.Password,
		http:     &http.Client{Timeout: requestTimeout, Transport: transport},
	}
}

func schemeOrDefault(scheme string) string {
	if scheme == "http" {
		return "http"
	}

	return "https"
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) Put(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

func (c *Client) Patch(ctx context.Context, path string, body any) error {
	return c.do(ctx, http.MethodPatch, path, body, nil)
}

func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}

		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("routeros %s %s: %w", method, path, err)
	}

	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return responseError(method, path, resp.StatusCode, data)
	}

	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}

	return json.Unmarshal(data, out)
}

func responseError(method, path string, status int, data []byte) error {
	var decoded apiError
	if json.Unmarshal(data, &decoded) == nil && decoded.Message != "" {
		message := decoded.Message
		if decoded.Detail != "" {
			message += ": " + decoded.Detail
		}

		return fmt.Errorf("routeros %s %s: HTTP %d: %s", method, path, status, message)
	}

	return fmt.Errorf("routeros %s %s: HTTP %d: %s", method, path, status, strings.TrimSpace(string(data)))
}
