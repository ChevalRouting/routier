package wireguard

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Generate a WireGuard keypair
// @Tags wireguard
// @Produce json
// @Success 200 {object} types.Response[types.KeygenResponse]
// @Security BearerAuth
// @Router /api/wireguard/keygen [post]
func Keygen(w http.ResponseWriter, _ *http.Request) {
	priv, pub, err := friends.GenerateWGKeyPair()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "key generation failed"))
		return
	}

	types.OK(w, types.KeygenResponse{PrivateKey: priv, PublicKey: pub})
}
