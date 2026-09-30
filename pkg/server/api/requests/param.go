package requests

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Parseable interface {
	~string | ~int | ~int64 | ~bool
}

func MustParam[T Parseable](r *http.Request, param string) (T, error) {
	var val T
	raw := chi.URLParam(r, param)
	if raw == "" {
		return val, errors.New("url parameter " + param + " is empty")
	}

	return parse[T](raw, val)
}

func QueryParam[T Parseable](r *http.Request, param string, def T) (T, error) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return def, nil
	}

	return parse[T](raw, def)
}

func parse[T Parseable](raw string, fallback T) (T, error) {
	switch any(fallback).(type) {
	case bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return fallback, err
		}

		return any(v).(T), nil
	case int:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return fallback, err
		}

		return any(v).(T), nil
	case int64:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fallback, err
		}

		return any(v).(T), nil
	default:
		return any(raw).(T), nil
	}
}
