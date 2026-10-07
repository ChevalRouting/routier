package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const v1MaxConfigBytes = 4 << 20

type v1ConfigDocument struct {
	YAML   string `json:"yaml"`
	SHA256 string `json:"sha256"`
}

func v1ConfigResponse(cfg *config.Config) (*v1ConfigDocument, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(data)
	return &v1ConfigDocument{YAML: string(data), SHA256: hex.EncodeToString(sum[:])}, nil
}

// @Summary Get the committed canonical configuration as YAML
// @Tags v1-config
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/config [get]
func handleV1GetConfig(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfgstore.Mu.RLock()
	cfg, err := config.Load(app.ConfigPath)
	cfgstore.Mu.RUnlock()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	doc, err := v1ConfigResponse(cfg)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to encode config"))
		return
	}

	types.OK(w, doc)
}

// @Summary Get a session's canonical configuration as YAML
// @Tags v1-config
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/config [get]
func handleV1GetSessionConfig(w http.ResponseWriter, r *http.Request) {
	doc, err := v1ConfigResponse(v1SessionFromCtx(r.Context()).Config)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to encode config"))
		return
	}

	types.OK(w, doc)
}

// @Summary Replace a session's canonical configuration with YAML
// @Tags v1-config
// @Accept plain
// @Produce json
// @Param sessionID path string true "session id"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/v1/sessions/{sessionID}/config [put]
func handleV1PutSessionConfig(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(io.LimitReader(r.Body, v1MaxConfigBytes+1))
	if err != nil {
		types.Err(http.StatusBadRequest, "failed to read config").Write(w)
		return
	}

	if len(data) > v1MaxConfigBytes {
		types.Err(http.StatusRequestEntityTooLarge, "config exceeds 4 MiB").Write(w)
		return
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		types.Err(http.StatusBadRequest, "invalid YAML: "+err.Error()).Write(w)
		return
	}

	if cfg.Version == "" {
		types.Err(http.StatusBadRequest, "config version is required").Write(w)
		return
	}

	app := appctx.FromContext(r.Context())
	sess := v1SessionFromCtx(r.Context())
	cfg.BaseDir = sess.BaseDir
	if err := v1WithLock(sess.ID, func() error {
		return webdb.UpdateSession(requests.DurableContext(r), app.DB, sess.ID, &cfg)
	}); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to save session"))
		return
	}

	doc, err := v1ConfigResponse(&cfg)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to encode config"))
		return
	}

	types.OK(w, doc)
}
