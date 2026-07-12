package nat

import (
	"encoding/json"
	"io"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	natpkg "github.com/ChevalRouting/routier/pkg/nat"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const maxRequestBodySize = 1 << 20

// List godoc
// @Summary  List NAT shortcut rules (masquerade/snat/dnat)
// @Tags nat
// @Produce json
// @Success 200 {object} types.Response[[]nat.Spec]
// @Security BearerAuth
// @Router /api/nat [get]
func List(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context()))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	specs := natpkg.Parse(cfg.Nftables)
	if specs == nil {
		specs = []natpkg.Spec{}
	}

	types.OK(w, specs)
}

// Replace godoc
// @Summary  Replace all NAT shortcut rules
// @Tags nat
// @Produce json
// @Param body body []nat.Spec true "NAT specs"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/nat [put]
func Replace(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	cfg, err := cfgstore.Read(app.ConfigPath, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		types.Err(http.StatusBadRequest, "failed to read request body").Write(w)
		return
	}

	var specs []natpkg.Spec
	if err := json.Unmarshal(body, &specs); err != nil {
		types.Error(log.Logger, w, types.Errorf(http.StatusBadRequest, "invalid nat data: %v", err))
		return
	}

	natpkg.Apply(cfg, specs)

	if appErr := cfgstore.ValidationError(cfg); appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	if err := cfgstore.WriteStaging(app.ConfigPath, username, cfg); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to write staging config"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
