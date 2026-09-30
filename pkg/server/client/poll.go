package client

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
)

func (c *Client) Poll(ctx context.Context, haveHash string) (*types.FriendPoll, error) {
	u := strings.TrimRight(c.gen.Server, "/") + "/api/friends/poll"
	if haveHash != "" {
		u += "?have=" + url.QueryEscape(haveHash)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	for _, edit := range c.gen.RequestEditors {
		if err := edit(ctx, req); err != nil {
			return nil, err
		}
	}

	resp, err := c.gen.Client.Do(req)
	return result[types.FriendPoll](c, resp, err, http.MethodGet, "/api/friends/poll")
}
