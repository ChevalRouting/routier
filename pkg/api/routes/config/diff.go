package config

import (
	"net/http"
	"os"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/diffutil"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// GetDiff godoc
// @Summary  Diff staged vs live config
// @Tags config
// @Produce json
// @Success 200 {object} types.Response[[]types.DiffLine]
// @Security BearerAuth
// @Router /api/config/diff [get]
func GetDiff(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	cfgstore.Mu.RLock()
	currentData, err := os.ReadFile(app.ConfigPath)
	cfgstore.Mu.RUnlock()
	if err != nil && !os.IsNotExist(err) {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "read config"))
		return
	}

	cfgstore.Mu.RLock()
	stagedData, err := os.ReadFile(cfgstore.StagingPath(app.ConfigPath, username))
	cfgstore.Mu.RUnlock()
	if err != nil && !os.IsNotExist(err) {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "read staging"))
		return
	}

	if os.IsNotExist(err) {
		stagedData = currentData
	}

	lines, err := diffutil.YAML(currentData, stagedData)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "compare config"))
		return
	}

	types.OK(w, lines)
}
