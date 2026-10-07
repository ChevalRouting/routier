package auth

import (
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/ChevalRouting/routier/pkg/auth/unixauth"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/host/motd"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

const seedPasswordFile = "/var/lib/routier/ui-seed-password"

// @Summary  Change the current user's password
// @Tags auth
// @Produce json
// @Param body body types.ChangePasswordRequest true "current and new password"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/auth/password [put]
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username, err := usernameFromRequest(r, app.JWTSecret)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	var req types.ChangePasswordRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	if !isUIAdmin(cfg, username) {
		types.Err(http.StatusUnauthorized, "user not found").Write(w)
		return
	}

	ok, err := unixauth.Verify(username, req.Current)
	if err != nil {
		log.Error().Err(err).Str("user", username).Msg("current password verification failed")
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to verify current password"))
		return
	}

	if !ok {
		log.Warn().Str("user", username).Msg("password change rejected because current password did not verify")
		types.Err(http.StatusUnauthorized, "current password is incorrect").Write(w)
		return
	}

	if err := setUnixPassword(username, req.New); err != nil {
		log.Error().Err(err).Str("user", username).Msg("failed to update Unix password")
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to update password"))
		return
	}

	log.Info().Str("user", username).Msg("Unix password updated")

	_ = os.Remove(seedPasswordFile)
	motd.Write(r.Context(), "", "")

	types.OK(w, types.StatusResponse{Status: "ok"})
}

func setUnixPassword(username, password string) error {
	cmd := exec.Command("chpasswd")
	cmd.Stdin = strings.NewReader(username + ":" + password + "\n")
	return cmd.Run()
}

func usernameFromRequest(r *http.Request, jwtSecret []byte) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", types.Errorf(http.StatusUnauthorized, "unauthorized")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenStr, appctx.JWTKeyFunc(jwtSecret))
	if err != nil || !token.Valid {
		return "", types.Errorf(http.StatusUnauthorized, "unauthorized")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", types.Errorf(http.StatusUnauthorized, "unauthorized")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", types.Errorf(http.StatusUnauthorized, "missing subject in token")
	}

	return sub, nil
}
