package types

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

type Response[T any] struct {
	Err            error `json:"-"`
	HTTPStatusCode int   `json:"-"`

	StatusText string `json:"status,omitempty" validate:"optional"`
	AppCode    int64  `json:"code,omitempty" validate:"optional"`
	ErrorText  string `json:"error,omitempty" validate:"optional"`
	Result     *T     `json:"result,omitempty" validate:"optional"`
}

func (e *Response[T]) Error() string {
	if e == nil {
		return ""
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	if e.ErrorText != "" {
		return e.ErrorText
	}

	return e.StatusText
}

func (e *Response[T]) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	if e.HTTPStatusCode != 0 {
		w.WriteHeader(e.HTTPStatusCode)
	}

	_ = json.NewEncoder(w).Encode(e)
}

func OK[T any](w http.ResponseWriter, data T) {
	(&Response[T]{HTTPStatusCode: http.StatusOK, Result: &data}).Write(w)
}

type Renderer interface {
	error
	Envelope() *Response[any]
}

func (e *Response[T]) Envelope() *Response[any] {
	return &Response[any]{
		HTTPStatusCode: e.HTTPStatusCode,
		StatusText:     e.StatusText,
		AppCode:        e.AppCode,
		ErrorText:      e.ErrorText,
	}
}

func Error(logger zerolog.Logger, w http.ResponseWriter, err error) {
	if rnd, ok := err.(Renderer); ok {
		rnd.Envelope().Write(w)
		return
	}

	logger.Err(err).Msg("500: internal server error")
	ErrUnexpected.Write(w)
}

var (
	ErrNotFound     = &Response[any]{HTTPStatusCode: http.StatusNotFound, ErrorText: "resource not found"}
	ErrUnauthorized = &Response[any]{HTTPStatusCode: http.StatusUnauthorized, ErrorText: "unauthorized"}
	ErrForbidden    = &Response[any]{HTTPStatusCode: http.StatusForbidden, ErrorText: "permission denied"}
	ErrBadRequest   = &Response[any]{HTTPStatusCode: http.StatusBadRequest, ErrorText: "bad request"}
	ErrUnexpected   = &Response[any]{HTTPStatusCode: http.StatusInternalServerError, ErrorText: "internal server error"}
)

func Err(code int, msg string) *Response[any] {
	return &Response[any]{HTTPStatusCode: code, ErrorText: msg}
}
