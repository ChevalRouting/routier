package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/ChevalRouting/routier/pkg/types"
)

var ErrUnknownDriver = errors.New("unknown switch driver")

type Target struct {
	Host          string
	Scheme        string
	Port          int
	TLSSkipVerify bool
	Credentials   map[string]string
}

type Step interface {
	Describe() string
	Run(ctx context.Context) error
}

type Driver interface {
	State(ctx context.Context) (types.SwitchState, error)
	Plan(ctx context.Context, plan types.SwitchPlan) ([]Step, error)
}

type Factory func(target Target) (Driver, error)

type Registry map[string]Factory

func (r Registry) New(name string, target Target) (Driver, error) {
	factory, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownDriver, name)
	}

	return factory(target)
}
