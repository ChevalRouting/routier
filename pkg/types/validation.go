package types

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag"`
	Value   any    `json:"value,omitempty" validate:"optional"`
}

type ValidationError struct {
	HTTPStatusCode int          `json:"-"`
	StatusText     string       `json:"status,omitempty" validate:"optional"`
	AppCode        int64        `json:"code,omitempty" validate:"optional"`
	ErrorText      string       `json:"error,omitempty" validate:"optional"`
	Fields         []FieldError `json:"fields,omitempty" validate:"optional"`
}

func (ve *ValidationError) Error() string {
	if ve.ErrorText != "" {
		return ve.ErrorText
	}

	return ve.StatusText
}

func (ve *ValidationError) Envelope() *Response[any] {
	return &Response[any]{
		HTTPStatusCode: ve.HTTPStatusCode,
		StatusText:     ve.StatusText,
		AppCode:        ve.AppCode,
		ErrorText:      ve.ErrorText,
	}
}

func (ve *ValidationError) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	if ve.HTTPStatusCode != 0 {
		w.WriteHeader(ve.HTTPStatusCode)
	}

	_ = json.NewEncoder(w).Encode(ve)
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email"
	case "min":
		return "value is too short"
	case "max":
		return "value is too long"
	case "oneof":
		return "invalid value"
	default:
		return "invalid value"
	}
}

func MapValidationErrors(ve validator.ValidationErrors) []FieldError {
	out := make([]FieldError, 0, len(ve))
	for _, fe := range ve {
		out = append(out, FieldError{
			Field:   fe.Field(),
			Tag:     fe.Tag(),
			Value:   fe.Value(),
			Message: validationMessage(fe),
		})
	}

	return out
}
