package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const maxResponseBytes = 32 << 20

type responseEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  any             `json:"error"`
}

func (i *instance) request(ctx context.Context, method, path, contentType string, body []byte) (any, error) {
	req, err := http.NewRequestWithContext(ctx, method, i.url+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+i.token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := i.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i.name, err)
	}

	defer func(action func() error) { _ = action() }(resp.Body.Close)
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}

	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("%s: response exceeds %d bytes", i.name, maxResponseBytes)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: HTTP %d: %s", i.name, resp.StatusCode, string(data))
	}

	var env responseEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("%s: decode response: %w", i.name, err)
	}

	var result any
	if len(env.Result) > 0 && string(env.Result) != "null" {
		if err := json.Unmarshal(env.Result, &result); err != nil {
			return nil, fmt.Errorf("%s: decode result: %w", i.name, err)
		}
	}

	return result, nil
}
