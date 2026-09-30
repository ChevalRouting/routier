package v1

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/diffutil"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type v1Session = webdb.Session

type v1SessionInfo = webdb.SessionInfo

var v1SessionLocks sync.Map

func v1WithLock(id string, fn func() error) error {
	v, _ := v1SessionLocks.LoadOrStore(id, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	return fn()
}

func v1NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

type v1SessCtxKey struct{}

func v1SessionFromCtx(ctx context.Context) *v1Session {
	v, _ := ctx.Value(v1SessCtxKey{}).(*v1Session)
	return v
}

func v1SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := appctx.FromContext(r.Context())
		id := chi.URLParam(r, "sessionID")
		username := appctx.UsernameFromContext(r.Context())

		sess, err := webdb.LoadSession(r.Context(), app.DB, id)
		if err != nil {
			types.Err(http.StatusInternalServerError, "failed to load session").Write(w)
			return
		}

		if sess == nil {
			types.Err(http.StatusNotFound, "session not found").Write(w)
			return
		}

		if sess.Username != username {
			types.Err(http.StatusForbidden, "session belongs to another user").Write(w)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), v1SessCtxKey{}, sess)))
	})
}

// handleV1CreateSession godoc
// @Summary  Create a config edit session
// @Tags v1-sessions
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions [post]
func handleV1CreateSession(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	cfgstore.Mu.RLock()
	cfg, err := config.Load(app.ConfigPath)
	cfgstore.Mu.RUnlock()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	id := v1NewID()
	createdAt := time.Now().UTC()
	if err := webdb.InsertSession(requests.DurableContext(r), app.DB, id, username, createdAt, cfg.BaseDir, cfg); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to create session"))
		return
	}

	types.OK(w, v1SessionInfo{ID: id, Username: username, CreatedAt: createdAt})
}

// handleV1ListSessions godoc
// @Summary  List the current user's sessions
// @Tags v1-sessions
// @Produce json
// @Success 200 {array} object
// @Security BearerAuth
// @Router /api/v1/sessions [get]
func handleV1ListSessions(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())
	list, err := webdb.ListUserSessions(r.Context(), app.DB, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to list sessions"))
		return
	}

	types.OK(w, list)
}

// handleV1GetSession godoc
// @Summary  Get a session
// @Tags v1-sessions
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID} [get]
func handleV1GetSession(w http.ResponseWriter, r *http.Request) {
	sess := v1SessionFromCtx(r.Context())
	types.OK(w, v1SessionInfo{ID: sess.ID, Username: sess.Username, CreatedAt: sess.CreatedAt})
}

// handleV1DeleteSession godoc
// @Summary  Discard a session
// @Tags v1-sessions
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID} [delete]
func handleV1DeleteSession(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	sess := v1SessionFromCtx(r.Context())
	if err := v1WithLock(sess.ID, func() error {
		return webdb.DeleteSession(requests.DurableContext(r), app.DB, sess.ID)
	}); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to delete session"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "discarded"})
}

// handleV1SessionDiff godoc
// @Summary  Diff a session vs live config
// @Tags v1-sessions
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/diff [get]
func handleV1SessionDiff(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	sess := v1SessionFromCtx(r.Context())

	cfgstore.Mu.RLock()
	currentData, err := os.ReadFile(app.ConfigPath)
	cfgstore.Mu.RUnlock()
	if err != nil && !os.IsNotExist(err) {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "read config"))
		return
	}

	sessData, err := yaml.Marshal(sess.Config)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "marshal session config"))
		return
	}

	lines := diffutil.Lines(string(currentData), string(sessData))

	types.OK(w, lines)
}

// handleV1SessionValidate godoc
// @Summary  Validate a session
// @Tags v1-sessions
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/validate [post]
func handleV1SessionValidate(w http.ResponseWriter, r *http.Request) {
	sess := v1SessionFromCtx(r.Context())
	if appErr := cfgstore.ValidationError(sess.Config); appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}

// handleV1SessionApply godoc
// @Summary  Apply a session to the live config
// @Tags v1-sessions
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/apply [post]
func handleV1SessionApply(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	sess := v1SessionFromCtx(r.Context())

	var result types.ApplyResult
	if err := v1WithLock(sess.ID, func() error {
		current, err := webdb.LoadSession(r.Context(), app.DB, sess.ID)
		if err != nil || current == nil {
			return types.NewError(http.StatusNotFound, "session not found or expired")
		}

		current.Config.BaseDir = current.BaseDir

		resolved, err := cfgstore.Resolve(current.Config)
		if err != nil {
			return types.Errorf(http.StatusBadRequest, "config interpolation: %v", err)
		}

		if errs := config.Validate(resolved, true); len(errs) > 0 {
			msgs := make([]string, len(errs))
			for i, e := range errs {
				msgs[i] = e.Error()
			}

			return types.Errorf(http.StatusBadRequest, "validation failed: %s", strings.Join(msgs, "; "))
		}

		cfgstore.Mu.RLock()
		oldData, _ := os.ReadFile(app.ConfigPath)
		cfgstore.Mu.RUnlock()

		newData, err := yaml.Marshal(current.Config)
		if err != nil {
			return types.Wrap(http.StatusInternalServerError, err, "marshal config")
		}

		cfgstore.Mu.Lock()
		if err := os.MkdirAll(filepath.Dir(app.ConfigPath), 0755); err != nil {
			cfgstore.Mu.Unlock()
			return types.Wrap(http.StatusInternalServerError, err, "create config dir")
		}

		if err := os.WriteFile(app.ConfigPath, newData, 0600); err != nil {
			cfgstore.Mu.Unlock()
			return types.Wrap(http.StatusInternalServerError, err, "write config")
		}

		cfgstore.Mu.Unlock()

		res, err := managers.Apply(r.Context(), resolved, friendcache.InterpolationVars(),
			managers.ApplyOptions{Source: "web", ConfigPath: app.ConfigPath}, managers.WatchdogTimeout)
		if err != nil {
			cfgstore.Mu.Lock()
			if len(oldData) > 0 {
				_ = os.WriteFile(app.ConfigPath, oldData, 0600)
			}

			cfgstore.Mu.Unlock()
			return types.Wrap(http.StatusInternalServerError, err, "apply failed (rolled back)")
		}

		_ = webdb.DeleteSession(requests.DurableContext(r), app.DB, sess.ID)

		result.Status = "applied"
		result.SnapID = res.SnapID
		result.Warning = res.Warning
		return nil
	}); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, result)
}
