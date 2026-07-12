package app

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-playground/validator/v10"
)

func NewValidator() *validator.Validate {
	v := validator.New()
	v.RegisterAlias("optional", "omitempty")
	return v
}

func (a *App) DecodeAndValidate(r *http.Request, dst any) *types.ValidationError {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		ve := &types.ValidationError{
			HTTPStatusCode: http.StatusBadRequest,
			AppCode:        1001,
			StatusText:     "bad json value",
		}

		if a.Debug {
			ve.ErrorText = err.Error()
		}

		return ve
	}

	v := a.Validator
	if v == nil {
		v = NewValidator()
	}

	if err := v.StructCtx(r.Context(), dst); err != nil {
		ve := &types.ValidationError{
			HTTPStatusCode: http.StatusUnprocessableEntity,
			AppCode:        1002,
			StatusText:     "validation error",
		}

		var fieldErrs validator.ValidationErrors
		if errors.As(err, &fieldErrs) {
			ve.Fields = types.MapValidationErrors(fieldErrs)
			return ve
		}

		if a.Debug {
			ve.ErrorText = err.Error()
		}

		return ve
	}

	return nil
}
