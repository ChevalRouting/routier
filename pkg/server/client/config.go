package client

import (
	"context"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
)

func (c *Client) Config(ctx context.Context) (*config.Config, error) {
	resp, err := c.gen.GetApiConfig(ctx, nil)
	return result[config.Config](c, resp, err, http.MethodGet, "/api/config")
}
