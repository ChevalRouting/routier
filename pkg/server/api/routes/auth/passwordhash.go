package auth

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/ChevalRouting/routier/pkg/unixauth"
	"github.com/rs/zerolog/log"
)

// HashPassword godoc
// @Summary  Hash a plaintext password for a config user (sha512-crypt)
// @Tags auth
// @Produce json
// @Param body body types.PasswordHashRequest true "password"
// @Success 200 {object} types.Response[types.PasswordHashResponse]
// @Security BearerAuth
// @Router /api/auth/password-hash [post]
func HashPassword(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	var req types.PasswordHashRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

	hash, err := unixauth.Hash(req.Password)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to hash password"))
		return
	}

	types.OK(w, types.PasswordHashResponse{Hash: hash})
}
