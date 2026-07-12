package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/types"
)

func (c *Client) Pair(ctx context.Context, req *types.FriendPairRequest) (*types.FriendPairResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.gen.PostApiFriendsPairWithBody(ctx, "application/json", bytes.NewReader(body))
	return result[types.FriendPairResponse](c, resp, err, http.MethodPost, "/api/friends/pair")
}
