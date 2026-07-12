package auth

import (
	"net/http"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/motd"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

// ChangePassword godoc
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

	user, err := webdb.UserByUsername(app.DB, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "database error"))
		return
	}

	if user == nil {
		types.Err(http.StatusUnauthorized, "user not found").Write(w)
		return
	}

	if !webdb.CheckPassword(user.PasswordHash, req.Current) {
		types.Err(http.StatusUnauthorized, "current password is incorrect").Write(w)
		return
	}

	newHash, err := webdb.HashPassword(req.New)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to hash password"))
		return
	}

	if err := webdb.UpdatePassword(app.DB, username, newHash); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to update password"))
		return
	}

	_ = webdb.SetSetting(app.DB, webdb.SettingPasswordChanged, "true")
	motd.Write("", "")

	types.OK(w, types.StatusResponse{Status: "ok"})
}

func usernameFromRequest(r *http.Request, jwtSecret []byte) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return "", types.Errorf(http.StatusUnauthorized, "missing or invalid authorization header")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenStr, appctx.JWTKeyFunc(jwtSecret))
	if err != nil || !token.Valid {
		return "", types.Errorf(http.StatusUnauthorized, "invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", types.Errorf(http.StatusUnauthorized, "invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", types.Errorf(http.StatusUnauthorized, "missing subject in token")
	}

	return sub, nil
}
