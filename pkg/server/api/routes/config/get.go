package config

import (
	"encoding/json"
	"fmt"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// GetConfig godoc
// @Summary  Get the staged config
// @Tags config
// @Produce json
// @Param layer query string false "configuration layer" Enums(advanced, simple)
// @Success 200 {object} types.Response[config.Config]
// @Security BearerAuth
// @Router /api/config [get]
func GetConfig(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	layer, appErr := requestLayer(r)
	if appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	cfg, err := cfgstore.ReadLayer(app.ConfigPath, appctx.UsernameFromContext(r.Context()), layer.Name())
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	if name, isFriend := appctx.FriendUsername(r.Context()); isFriend {
		f := friendspkg.Get(cfg, name)
		if app.Identity == nil || f == nil || f.Identity.X25519PublicKey == "" {
			types.Err(http.StatusConflict, "friend \""+name+"\" not paired for encryption").Write(w)
			return
		}

		sealed, serr := sealForFriend(app, cfg, name, cfg)
		if serr != nil {
			types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, serr, "seal config for friend"))
			return
		}

		w.Header().Set("Content-Type", types.SealedContentType)
		w.Write(sealed)
		return
	}

	projected, err := layer.Project(cfg)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to project config"))
		return
	}

	if layer.Name() == "advanced" {
		types.OK(w, cfg)
		return
	}

	types.OK(w, projected)
}

func sealForFriend(app *appctx.App, cfg *cfgpkg.Config, friendName string, payload any) ([]byte, error) {
	if app.Identity == nil {
		return nil, fmt.Errorf("no local identity")
	}

	f := friendspkg.Get(cfg, friendName)
	if f == nil || f.Identity.X25519PublicKey == "" {
		return nil, fmt.Errorf("friend %q not paired for encryption", friendName)
	}

	env, err := json.Marshal(types.Response[any]{Result: &payload})
	if err != nil {
		return nil, err
	}

	return app.Identity.Seal(f.Identity.X25519PublicKey, env)
}
