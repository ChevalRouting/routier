package setup

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	setuppkg "github.com/ChevalRouting/routier/pkg/setup"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Stage the first-run network: hostname, Internet uplink and local networks
// @Tags setup
// @Produce json
// @Param body body setup.Network true "network"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/setup/network [put]
func PutNetwork(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	var network setuppkg.Network
	if ve := app.DecodeAndValidate(r, &network); ve != nil {
		ve.Write(w)
		return
	}

	username := appctx.UsernameFromContext(r.Context())
	cfg, err := cfgstore.Read(app.ConfigPath, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	if err := setuppkg.Apply(cfg, network); err != nil {
		types.Err(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	if appErr := cfgstore.ValidationError(cfg); appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	if err := cfgstore.WriteStaging(app.ConfigPath, username, cfg); err != nil {
		types.Error(log.Logger, w, cfgstore.StagingError(err, "failed to write staging config"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
