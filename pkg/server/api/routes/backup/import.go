package backup

import (
	"io"
	"net/http"
	"os"

	"github.com/ChevalRouting/routier/pkg/managers"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	backuppkg "github.com/ChevalRouting/routier/pkg/state/backup"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Restore a config backup and apply it
// @Tags backup
// @Produce json
// @Success 200 {object} types.Response[types.ApplyResult]
// @Security BearerAuth
// @Router /api/backup/import [post]
func Import(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		types.Err(http.StatusBadRequest, "invalid upload").Write(w)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		types.Err(http.StatusBadRequest, "missing 'file' field").Write(w)
		return
	}

	defer func(action func() error) { _ = action() }(file.Close)

	tmp, err := os.CreateTemp("", "routier-import-*.bin")
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to stage upload"))
		return
	}

	tmpPath := tmp.Name()
	defer func(path string) { _ = os.Remove(path) }(tmpPath)
	if _, err := io.Copy(tmp, io.LimitReader(file, 256<<20)); err != nil {
		_ = tmp.Close()
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to stage upload"))
		return
	}

	_ = tmp.Close()

	restored, err := backuppkg.Restore(tmpPath, app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "invalid backup archive"))
		return
	}

	cfg, err := restored.Load()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "restored config is invalid"))
		return
	}

	res, err := managers.Apply(r.Context(), cfg, friendcache.InterpolationVars(),
		managers.ApplyOptions{Source: "restore", ConfigPath: restored.ConfigPath, PersistConfig: restored.Config}, managers.WatchdogTimeout)
	if err != nil {
		types.Error(log.Logger, w, cfgstore.ApplyError(err, http.StatusInternalServerError, "apply failed (rolled back)"))
		return
	}

	types.OK(w, types.ApplyResult{Status: "applied", SnapID: res.SnapID, Warning: res.Warning})
}
