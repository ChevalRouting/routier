package auth

import (
	"net/http"
	"sync"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

type loginLimiterState struct {
	mu          sync.Mutex
	fails       map[string]int
	lockedUntil map[string]time.Time
}

var loginLimiter = loginLimiterState{
	fails:       make(map[string]int),
	lockedUntil: make(map[string]time.Time),
}

func checkLoginAllowed(username string) bool {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()
	if until, ok := loginLimiter.lockedUntil[username]; ok && time.Now().Before(until) {
		return false
	}

	return true
}

func recordLoginFailure(username string) {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()
	loginLimiter.fails[username]++
	if loginLimiter.fails[username] >= 5 {
		loginLimiter.lockedUntil[username] = time.Now().Add(30 * time.Second)
		loginLimiter.fails[username] = 0
	}
}

func recordLoginSuccess(username string) {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()
	delete(loginLimiter.fails, username)
	delete(loginLimiter.lockedUntil, username)
}

// Login godoc
// @Summary  Authenticate and obtain a token
// @Tags auth
// @Produce json
// @Param body body types.LoginRequest true "credentials"
// @Success 200 {object} types.Response[types.LoginResponse]
// @Router /api/auth/login [post]
func Login(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	var req types.LoginRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

	if !checkLoginAllowed(req.Username) {
		types.Err(http.StatusTooManyRequests, "too many failed attempts, try again later").Write(w)
		return
	}

	user, err := webdb.UserByUsername(app.DB, req.Username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "database error"))
		return
	}

	if user == nil || !webdb.CheckPassword(user.PasswordHash, req.Password) {
		recordLoginFailure(req.Username)
		types.Err(http.StatusUnauthorized, "invalid credentials").Write(w)
		return
	}

	recordLoginSuccess(req.Username)

	claims := jwt.MapClaims{
		"sub": user.Username,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(app.JWTSecret)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to sign token"))
		return
	}

	types.OK(w, types.LoginResponse{Token: signed})
}
