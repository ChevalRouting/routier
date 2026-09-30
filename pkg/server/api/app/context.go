package app

import (
	"context"
	"net/http"
	"strings"
)

const FriendUserPrefix = "_friend:"

type ctxKey string

const (
	appKey      ctxKey = "app"
	usernameKey ctxKey = "username"
)

func FromContext(ctx context.Context) *App {
	v, _ := ctx.Value(appKey).(*App)
	return v
}

func UsernameFromContext(ctx context.Context) string {
	v, _ := ctx.Value(usernameKey).(string)
	return v
}

func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey, username)
}

func FriendUsername(ctx context.Context) (string, bool) {
	u := UsernameFromContext(ctx)
	if name, ok := strings.CutPrefix(u, FriendUserPrefix); ok {
		return name, true
	}

	return "", false
}

func (a *App) Inject() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), appKey, a)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
