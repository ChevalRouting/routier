package auth

import (
	"net/http"
	"sync"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/ChevalRouting/routier/pkg/unixauth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"slices"
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
		log.Warn().Msg("login request validation failed")
		ve.Write(w)
		return
	}

	log.Info().Str("user", req.Username).Msg("login attempt")
	if !checkLoginAllowed(req.Username) {
		log.Warn().Str("user", req.Username).Msg("login rejected by rate limiter")
		types.Err(http.StatusTooManyRequests, "too many failed attempts, try again later").Write(w)
		return
	}

	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	if !isUIAdmin(cfg, req.Username) {
		recordLoginFailure(req.Username)
		log.Warn().Str("user", req.Username).Msg("login rejected because user is not a UI administrator")
		types.Err(http.StatusUnauthorized, "invalid credentials").Write(w)
		return
	}

	ok, err := unixauth.Verify(req.Username, req.Password)
	if err != nil {
		log.Error().Err(err).Str("user", req.Username).Msg("login password verification failed")
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to verify password"))
		return
	}
	if !ok {
		recordLoginFailure(req.Username)
		log.Warn().Str("user", req.Username).Msg("login rejected because password did not verify")
		types.Err(http.StatusUnauthorized, "invalid credentials").Write(w)
		return
	}

	recordLoginSuccess(req.Username)
	log.Info().Str("user", req.Username).Msg("login authenticated")

	claims := jwt.MapClaims{
		"sub": req.Username,
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

const builtinAdmin = "routier"

func isUIAdmin(cfg *config.Config, username string) bool {
	if username == "" || username == "root" {
		return false
	}

	if username == builtinAdmin {
		return true
	}

	u := cfg.Users[username]
	return u != nil && slices.Contains(u.Groups, "wheel")
}
