package wireguard

import (
	"encoding/base64"
	"io"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/types"
	"golang.org/x/crypto/curve25519"
)

// @Summary  Derive the public key for a private key
// @Tags wireguard
// @Accept plain
// @Produce json
// @Success 200 {object} types.Response[types.PubKeyResponse]
// @Security BearerAuth
// @Router /api/wireguard/pubkey [post]
func Pubkey(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 256))
	if err != nil || len(body) == 0 {
		types.Err(http.StatusBadRequest, "missing private key in body").Write(w)
		return
	}

	privBytes, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil || len(privBytes) != 32 {
		types.Err(http.StatusBadRequest, "invalid private key (must be 32-byte base64)").Write(w)
		return
	}

	pub, err := curve25519.X25519(privBytes, curve25519.Basepoint)
	if err != nil {
		types.Err(http.StatusBadRequest, "key derivation failed: "+err.Error()).Write(w)
		return
	}

	types.OK(w, types.PubKeyResponse{PublicKey: base64.StdEncoding.EncodeToString(pub)})
}
