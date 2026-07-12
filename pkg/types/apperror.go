package types

import "fmt"

type AppError struct {
	Code    int
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func (e *AppError) Envelope() *Response[any] {
	msg := e.Message
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}

	return &Response[any]{HTTPStatusCode: e.Code, AppCode: int64(e.Code), ErrorText: msg}
}

func NewError(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func Errorf(code int, format string, args ...any) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func Wrap(code int, err error, msg string) *AppError {
	return &AppError{Code: code, Message: msg, Cause: err}
}

type ErrorDetail struct {
	Trace string `json:"trace,omitempty" validate:"optional"`
}
