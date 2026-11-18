package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/managers"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/types"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/golang-jwt/jwt/v5"
)

type ctxQueryTokenKey struct{}

func redactTokenQueryParam(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveWithRedactedToken(next, w, r) })
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveCORS(next, w, r) })
}

type stagingWriter struct {
	http.ResponseWriter
	configPath string
	username   string
	headerDone bool
}

func (sw *stagingWriter) setStagingHeader() {
	if sw.headerDone {
		return
	}

	sw.headerDone = true
	pending := "false"

	if staged, err := os.ReadFile(cfgstore.StagingPath(sw.configPath, sw.username)); err == nil {
		committed, ok := managers.PendingConfig(sw.configPath)
		if !ok {
			committed, _ = os.ReadFile(sw.configPath)
		}

		if !bytes.Equal(staged, committed) {
			pending = "true"
		}
	}

	sw.ResponseWriter.Header().Set("X-Staging-Pending", pending)
}

func (sw *stagingWriter) WriteHeader(code int) {
	sw.setStagingHeader()
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *stagingWriter) Write(b []byte) (int, error) {
	sw.setStagingHeader()
	return sw.ResponseWriter.Write(b)
}

func (sw *stagingWriter) Flush() {
	if f, ok := sw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (sw *stagingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := sw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("websocket: underlying ResponseWriter does not implement http.Hijacker")
	}

	return h.Hijack()
}

func stagingHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveWithStagingHeaders(next, w, r) })
}

func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { authenticate(next, w, r) })
}

func friendTokenAllowed(f *config.Friend, method, path string) bool {
	if f.Manage {
		return strings.HasPrefix(path, "/api/")
	}

	switch {
	case method == http.MethodGet && path == "/api/friends/hello":
		return true
	case method == http.MethodGet && path == "/api/friends/poll":
		return true
	case method == http.MethodGet && path == "/api/friends/interfaces":
		return true
	case method == http.MethodGet && path == "/api/friends/exports":
		return true
	case method == http.MethodGet && path == "/api/config":
		return true
	case method == http.MethodGet && path == "/api/apply/pending":
		return true
	case method == http.MethodPost && path == "/api/friends/pair":
		return true
	case method == http.MethodPut && path == "/api/config/import":
		return true
	case method == http.MethodPost && path == "/api/config/apply":
		return true
	case method == http.MethodPost && path == "/api/apply/confirm":
		return true
	}

	return false
}

func serveWithRedactedToken(next http.Handler, w http.ResponseWriter, r *http.Request) {
	if tok := r.URL.Query().Get("token"); tok != "" {
		ctx := context.WithValue(r.Context(), ctxQueryTokenKey{}, tok)
		q := r.URL.Query()
		q.Set("token", "REDACTED")
		r2 := r.Clone(ctx)
		r2.URL.RawQuery = q.Encode()
		next.ServeHTTP(w, r2)
		return
	}

	next.ServeHTTP(w, r)
}

func serveCORS(next http.Handler, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "X-Staging-Pending")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	next.ServeHTTP(w, r)
}

func serveWithStagingHeaders(next http.Handler, w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	if username := appctx.UsernameFromContext(r.Context()); username != "" && app != nil {
		next.ServeHTTP(&stagingWriter{ResponseWriter: w, configPath: app.ConfigPath, username: username}, r)
		return
	}

	next.ServeHTTP(w, r)
}

func authenticate(next http.Handler, w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	if app == nil {
		types.Err(http.StatusInternalServerError, "missing app context").Write(w)
		return
	}

	var tokenStr string
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	} else if q, _ := r.Context().Value(ctxQueryTokenKey{}).(string); q != "" {
		tokenStr = q
	} else {
		types.Err(http.StatusUnauthorized, "unauthorized").Write(w)
		return
	}

	token, err := jwt.Parse(tokenStr, appctx.JWTKeyFunc(app.JWTSecret))
	if err == nil && token.Valid {
		claims, _ := token.Claims.(jwt.MapClaims)
		username, _ := claims["sub"].(string)
		if !friendTokens.adminAllowed(app.ConfigPath, username, time.Now()) {
			types.Err(http.StatusUnauthorized, "unauthorized").Write(w)
			return
		}

		ctx := appctx.WithUnixUser(appctx.WithUsername(r.Context(), username), username)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	if strings.HasPrefix(tokenStr, webdb.APIKeyPrefix) {
		key, keyErr := webdb.AuthenticateAPIKey(r.Context(), app.DB, tokenStr)
		if keyErr == nil {
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
				types.Err(http.StatusForbidden, "API keys cannot manage authentication credentials").Write(w)
				return
			}

			ctx := appctx.WithUnixUser(appctx.WithUsername(r.Context(), "apikey_"+key.ID), key.CreatedBy)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
	}

	if f := friendTokens.lookup(app.ConfigPath, tokenStr, time.Now()); f != nil {
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			types.Err(http.StatusForbidden, "friend tokens cannot manage authentication credentials").Write(w)
			return
		}

		if !friendTokenAllowed(f, r.Method, r.URL.Path) {
			types.Err(http.StatusForbidden, "friend token not permitted for this endpoint").Write(w)
			return
		}

		ctx := appctx.WithUsername(r.Context(), appctx.FriendUserPrefix+f.Name)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	types.Err(http.StatusUnauthorized, "unauthorized").Write(w)
}
