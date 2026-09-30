package client

import (
	"context"
	"net/http"

	routierclient "github.com/ChevalRouting/routier/pkg/server/client/generated"
	"github.com/ChevalRouting/routier/pkg/types"
)

func (c *Client) Hello(ctx context.Context) (*types.FriendsHello, error) {
	resp, err := c.gen.GetApiFriendsHello(ctx, &routierclient.GetApiFriendsHelloParams{})
	return result[types.FriendsHello](c, resp, err, http.MethodGet, "/api/friends/hello")
}

func (c *Client) HelloChallenge(ctx context.Context, challenge string) (*types.FriendsHello, error) {
	resp, err := c.gen.GetApiFriendsHello(ctx, &routierclient.GetApiFriendsHelloParams{Challenge: &challenge})
	return result[types.FriendsHello](c, resp, err, http.MethodGet, "/api/friends/hello")
}
