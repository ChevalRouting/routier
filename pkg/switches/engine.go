package switches

import (
	"context"
	"fmt"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/switches/driver"
	"github.com/ChevalRouting/routier/pkg/switches/driver/routeros"
	"github.com/ChevalRouting/routier/pkg/switches/model"
	"github.com/ChevalRouting/routier/pkg/types"
)

const DefaultDBPath = "/var/lib/routier/web.db"

var DefaultDrivers = driver.Registry{
	routeros.Name: routeros.New,
}

type Engine struct {
	creds   *Credentials
	catalog *model.Catalog
	drivers driver.Registry
}

type Logf func(format string, args ...any)

func NewEngine(creds *Credentials, catalog *model.Catalog, drivers driver.Registry) *Engine {
	return &Engine{creds: creds, catalog: catalog, drivers: drivers}
}

func OpenEngine(ctx context.Context, dbPath string) (*Engine, func(), error) {
	db, err := webdb.InitDB(ctx, dbPath)
	if err != nil {
		return nil, nil, err
	}

	closeDB := func() { _ = db.Close() }
	creds, err := OpenCredentials(db, dbPath)
	if err != nil {
		closeDB()
		return nil, nil, err
	}

	catalog, err := model.Default()
	if err != nil {
		closeDB()
		return nil, nil, err
	}

	return NewEngine(creds, catalog, DefaultDrivers), closeDB, nil
}

func (e *Engine) Credentials() *Credentials {
	return e.creds
}

func (e *Engine) Catalog() *model.Catalog {
	return e.catalog
}

func (e *Engine) Connect(ctx context.Context, name string, sw *config.Switch) (driver.Driver, error) {
	m, err := e.catalog.Get(sw.Model)
	if err != nil {
		return nil, err
	}

	creds, err := e.creds.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	return e.drivers.New(m.Driver, driver.Target{Host: sw.Host, Scheme: sw.Scheme, Port: sw.Port, TLSSkipVerify: sw.TLSSkipVerify, Credentials: creds})
}

func (e *Engine) State(ctx context.Context, name string, sw *config.Switch) (types.SwitchState, error) {
	d, err := e.Connect(ctx, name, sw)
	if err != nil {
		return types.SwitchState{}, err
	}

	return d.State(ctx)
}

func (e *Engine) Plan(ctx context.Context, name string, desired, previous *config.Switch) ([]driver.Step, error) {
	steps, _, err := e.plan(ctx, name, desired, previous)
	return steps, err
}

func (e *Engine) plan(ctx context.Context, name string, desired, previous *config.Switch) ([]driver.Step, types.SwitchState, error) {
	d, err := e.Connect(ctx, name, desired)
	if err != nil {
		return nil, types.SwitchState{}, err
	}

	live, err := d.State(ctx)
	if err != nil {
		return nil, types.SwitchState{}, err
	}

	changes, err := Changes(desired, previous, live)
	if err != nil || len(changes) == 0 {
		return nil, live, err
	}

	steps, err := d.Plan(ctx, types.SwitchPlan{Changes: changes})
	return steps, live, err
}

type BeforeChange func(live types.SwitchState) error

func (e *Engine) Reconcile(ctx context.Context, name string, desired, previous *config.Switch, dryRun bool, logf Logf, before BeforeChange) (int, error) {
	steps, live, err := e.plan(ctx, name, desired, previous)
	if err != nil {
		return 0, err
	}

	if len(steps) > 0 && !dryRun && before != nil {
		if err := before(live); err != nil {
			return 0, fmt.Errorf("save the state of the switch before changing it: %w", err)
		}
	}

	for index, step := range steps {
		if dryRun {
			logf("would: switch %s: %s", name, step.Describe())
			continue
		}

		logf("switch %s: %s", name, step.Describe())
		if err := step.Run(ctx); err != nil {
			return index, fmt.Errorf("step %d of %d failed: %w", index+1, len(steps), err)
		}
	}

	return len(steps), nil
}
