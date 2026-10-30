package switches

import (
	"context"
	"errors"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/switchcache"
	swengine "github.com/ChevalRouting/routier/pkg/switches"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
)

func Routes(r chi.Router) {
	r.Get("/api/switch-models", Models)
	r.Get("/api/switches", List)
	r.Get("/api/switches/candidates", Candidates)
	r.Get("/api/switches/{name}", Get)
	r.Post("/api/switches/{name}/refresh", Refresh)
	r.Get("/api/switches/{name}/plan", Plan)
	r.Put("/api/switches/{name}/credentials", SetCredentials)
	r.Delete("/api/switches/{name}/credentials", DeleteCredentials)
}

func engineError(err error, msg string) error {
	switch {
	case errors.Is(err, swengine.ErrNoCredentials):
		return types.NewError(http.StatusConflict, "no credentials stored for this switch")
	case errors.Is(err, swengine.ErrVLANFilteringOff):
		return types.Wrap(http.StatusConflict, err, msg)
	}

	return types.Wrap(http.StatusBadGateway, err, msg)
}

func status(ctx context.Context, app *appctx.App, name string, sw *config.Switch, names map[string]bool) types.SwitchStatus {
	result := types.SwitchStatus{Name: name, Host: sw.Host, Interface: sw.Interface, Uplink: sw.Uplink, HasCredentials: names[name]}
	if found, err := app.Switches.Catalog().Get(sw.Model); err == nil {
		result.Model = found
	} else {
		result.Model = types.SwitchModel{ID: sw.Model, Name: sw.Model}
	}

	if entry, ok := switchcache.Current(ctx, app.Switches, name, sw); ok {
		result.State, result.ReadAt, result.Error = entry.State, entry.ReadAt, entry.Error
		result.Reachable = entry.Error == "" && entry.State != nil
	}

	return result
}
