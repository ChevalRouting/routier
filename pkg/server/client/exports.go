package client

import (
	"context"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/types"
)

func (c *Client) Exports(ctx context.Context) ([]types.FriendVar, error) {
	resp, err := c.gen.GetApiFriendsExports(ctx)
	vars, err := result[[]types.FriendVar](c, resp, err, http.MethodGet, "/api/friends/exports")
	if err != nil {
		return nil, err
	}

	return *vars, nil
}
