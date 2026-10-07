package failures

import (
	"fmt"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
)

type ValidationError struct {
	BundleID string
	Errors   []types.ArtifactError
}

func NewValidationError(errs []types.ArtifactError) *ValidationError {
	return &ValidationError{Errors: errs}
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "artifact validation failed"
	}

	parts := make([]string, 0, len(e.Errors))
	for _, x := range e.Errors {
		loc := x.Dest
		if x.Line > 0 {
			loc = fmt.Sprintf("%s:%d", x.Dest, x.Line)
		}

		parts = append(parts, strings.TrimSpace(strings.TrimPrefix(loc+": "+x.Message, ": ")))
	}

	return "artifact validation failed: " + strings.Join(parts, "; ")
}

type ApplyError struct {
	Cause    error
	LogID    string
	BundleID string
}

func (e *ApplyError) Error() string { return e.Cause.Error() }
func (e *ApplyError) Unwrap() error { return e.Cause }
