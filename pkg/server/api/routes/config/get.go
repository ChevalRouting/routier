package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Get the staged config
// @Tags config
// @Produce json
// @Param source query string false "staging (the caller's staged copy when it has one) or running" Enums(staging, running)
// @Success 200 {object} types.Response[config.Config]
// @Security BearerAuth
// @Router /api/config [get]
func GetConfig(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := readSource(r, app.ConfigPath)
	if errors.Is(err, errUnknownSource) {
		types.Error(log.Logger, w, types.Errorf(http.StatusBadRequest, "unknown config source: %s", r.URL.Query().Get("source")))
		return
	}

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
		_, _ = w.Write(sealed)
		return
	}

	types.OK(w, cfg)
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

var errUnknownSource = errors.New("unknown config source")

func readSource(r *http.Request, configPath string) (*cfgpkg.Config, error) {
	switch r.URL.Query().Get("source") {
	case "", "staging":
		return cfgstore.Read(configPath, appctx.UsernameFromContext(r.Context()))
	case "running":
		return cfgstore.ReadRunning(configPath)
	default:
		return nil, errUnknownSource
	}
}
