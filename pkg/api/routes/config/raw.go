package config

import (
	"io"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const maxRequestBodySize = 4 << 20

// PutRaw godoc
// @Summary  Replace the whole staged config (raw YAML)
// @Tags config
// @Accept plain
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/config/raw [put]
func PutRaw(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		types.Err(http.StatusBadRequest, "failed to read request body").Write(w)
		return
	}

	var cfg cfgpkg.Config
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		types.Err(http.StatusBadRequest, "invalid YAML: "+err.Error()).Write(w)
		return
	}

	if cfg.Version == "" {
		types.Err(http.StatusBadRequest, "config version is required").Write(w)
		return
	}

	if appErr := cfgstore.ValidationError(&cfg); appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	if err := cfgstore.WriteStaging(app.ConfigPath, username, &cfg); err != nil {
		types.Error(log.Logger, w, cfgstore.StagingError(err, "failed to write staging config"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
