package v1

import (
	"encoding/json"
	"net/http"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/requests"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func v1MapRoutes(r chi.Router, name string, list, replace, getOne, putOne, deleteOne http.HandlerFunc) {
	r.Get("/"+name, list)
	r.Put("/"+name, replace)
	r.Get("/"+name+"/{name}", getOne)
	r.Put("/"+name+"/{name}", putOne)
	r.Delete("/"+name+"/{name}", deleteOne)
}

func v1Mutate(r *http.Request, fn func(c *config.Config) error) error {
	app := appctx.FromContext(r.Context())
	sess := v1SessionFromCtx(r.Context())
	return v1WithLock(sess.ID, func() error {
		s, err := webdb.LoadSession(r.Context(), app.DB, sess.ID)
		if err != nil || s == nil {
			return types.NewError(http.StatusNotFound, "session not found")
		}

		before := validationSet(s.Config)
		if err := fn(s.Config); err != nil {
			return err
		}

		if appErr := newValidationErrors(before, s.Config); appErr != nil {
			return appErr
		}

		if err := webdb.UpdateSession(requests.DurableContext(r), app.DB, sess.ID, s.Config); err != nil {
			return types.Wrap(http.StatusInternalServerError, err, "failed to save session")
		}

		return nil
	})
}

func validationSet(cfg *config.Config) map[string]struct{} {
	return config.ValidationSet(cfg)
}

func newValidationErrors(before map[string]struct{}, cfg *config.Config) *types.AppError {
	added := config.NewValidationErrors(before, cfg)
	if len(added) == 0 {
		return nil
	}

	return types.Errorf(http.StatusBadRequest, "validation failed: %s", strings.Join(added, "; "))
}

func v1Decode[T any](r *http.Request) (T, error) {
	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return body, types.Errorf(http.StatusBadRequest, "invalid body: %v", err)
	}

	return body, nil
}

func v1GetObject[T any](w http.ResponseWriter, r *http.Request, get func(c *config.Config) T) {
	sess := v1SessionFromCtx(r.Context())
	types.OK(w, get(sess.Config))
}

func v1PutObject[T any](w http.ResponseWriter, r *http.Request, set func(c *config.Config, v T)) {
	body, err := v1Decode[T](r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := v1Mutate(r, func(c *config.Config) error { set(c, body); return nil }); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, body)
}

func mapOrEmpty[T any](m map[string]T) map[string]T {
	if m == nil {
		return map[string]T{}
	}

	return m
}

func v1MapGetKey[T any](w http.ResponseWriter, r *http.Request, get func(c *config.Config) map[string]T) {
	sess := v1SessionFromCtx(r.Context())
	name := chi.URLParam(r, "name")
	v, ok := get(sess.Config)[name]
	if !ok {
		types.Error(log.Logger, w, types.Errorf(http.StatusNotFound, "%q not found", name))
		return
	}

	types.OK(w, v)
}

func v1MapPutKey[T any](
	w http.ResponseWriter, r *http.Request,
	get func(c *config.Config) map[string]T,
	set func(c *config.Config, m map[string]T),
) {
	name := chi.URLParam(r, "name")
	body, err := v1Decode[T](r)
	if err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	if err := v1Mutate(r, func(c *config.Config) error {
		m := get(c)
		if m == nil {
			m = make(map[string]T)
			set(c, m)
		}

		m[name] = body
		return nil
	}); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, body)
}

func v1MapDeleteKey[T any](w http.ResponseWriter, r *http.Request, get func(c *config.Config) map[string]T) {
	name := chi.URLParam(r, "name")
	if err := v1Mutate(r, func(c *config.Config) error {
		m := get(c)
		if _, ok := m[name]; !ok {
			return types.Errorf(http.StatusNotFound, "%q not found", name)
		}

		delete(m, name)
		return nil
	}); err != nil {
		types.Error(log.Logger, w, err)
		return
	}

	types.OK(w, types.StatusResponse{Status: "deleted"})
}
