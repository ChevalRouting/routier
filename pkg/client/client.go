package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	routierclient "github.com/ChevalRouting/routier/pkg/client/generated"
	"github.com/ChevalRouting/routier/pkg/identity"
	"github.com/ChevalRouting/routier/pkg/types"
)

const defaultTimeout = 8 * time.Second
const maxResponseBytes = 32 << 20

type Config struct {
	URL           string
	Token         string
	TLSSkipVerify bool
	Timeout       time.Duration
}

type Client struct {
	gen       *routierclient.Client
	identity  *identity.Identity
	peerPub   string
	sharedKey [32]byte
	hasShared bool
}

func (c *Client) SetSealOpener(self *identity.Identity, peerEdPub string) {
	c.identity = self
	c.peerPub = peerEdPub
}

func (c *Client) SetSharedKey(key [32]byte) {
	c.sharedKey = key
	c.hasShared = true
}

func New(c *Config) *Client {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: c.TLSSkipVerify}

	httpc := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	auth := func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+c.Token)
		return nil
	}

	gen, _ := routierclient.NewClient(
		strings.TrimRight(c.URL, "/"),
		routierclient.WithHTTPClient(httpc),
		routierclient.WithRequestEditorFn(auth),
	)

	return &Client{gen: gen}
}

type ErrUnexpectedStatus struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e *ErrUnexpectedStatus) Error() string {
	return fmt.Sprintf("routier %s %s: HTTP %d: %s", e.Method, e.Path, e.StatusCode, strings.TrimSpace(e.Body))
}

func result[T any](c *Client, resp *http.Response, err error, method, path string) (*T, error) {
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("routier %s %s: response exceeds %d bytes", method, path, maxResponseBytes)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &ErrUnexpectedStatus{Method: method, Path: path, StatusCode: resp.StatusCode, Body: string(data)}
	}

	switch resp.Header.Get("Content-Type") {
	case types.SealedContentType:
		if c.identity == nil {
			return nil, fmt.Errorf("routier %s %s: got a sealed response but no decryption key configured", method, path)
		}

		plain, oerr := c.identity.Open(data, c.peerPub)
		if oerr != nil {
			return nil, fmt.Errorf("routier %s %s: %w", method, path, oerr)
		}

		data = plain
	case types.SharedSealedContentType:
		if !c.hasShared {
			return nil, fmt.Errorf("routier %s %s: got a shared-sealed response but no shared key configured", method, path)
		}

		plain, oerr := identity.OpenShared(c.sharedKey, data)
		if oerr != nil {
			return nil, fmt.Errorf("routier %s %s: %w", method, path, oerr)
		}

		data = plain
	}

	return unwrap[T](data)
}

func unwrap[T any](data []byte) (*T, error) {
	var wrapped types.Response[T]
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, err
	}

	if wrapped.Result == nil {
		return nil, fmt.Errorf("empty response")
	}

	return wrapped.Result, nil
}
