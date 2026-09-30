package client

import (
	"context"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/types"
)

func (c *Client) Interfaces(ctx context.Context) ([]types.FriendInterface, error) {
	resp, err := c.gen.GetApiFriendsInterfaces(ctx)
	ifaces, err := result[[]types.FriendInterface](c, resp, err, http.MethodGet, "/api/friends/interfaces")
	if err != nil {
		return nil, err
	}

	return *ifaces, nil
}
