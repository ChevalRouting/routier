package backup

import (
	"io"
	"net/http"
	"os"

	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	backuppkg "github.com/ChevalRouting/routier/pkg/backup"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Import godoc
// @Summary  Restore a config backup and apply it
// @Tags backup
// @Produce json
// @Success 200 {object} types.Response[types.ApplyResult]
// @Security BearerAuth
// @Router /api/backup/import [post]
func Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		types.Err(http.StatusBadRequest, "invalid upload").Write(w)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		types.Err(http.StatusBadRequest, "missing 'file' field").Write(w)
		return
	}

	defer file.Close()

	tmp, err := os.CreateTemp("", "routier-import-*.bin")
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to stage upload"))
		return
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, io.LimitReader(file, 256<<20)); err != nil {
		tmp.Close()
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to stage upload"))
		return
	}

	tmp.Close()

	cfgPath, err := backuppkg.Restore(tmpPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "invalid backup archive"))
		return
	}

	cfg, err := config.LoadAndValidate(cfgPath, true)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, err, "restored config is invalid"))
		return
	}

	res, err := managers.Apply(r.Context(), cfg, friendcache.InterpolationVars(),
		managers.ApplyOptions{Source: "restore", ConfigPath: cfgPath}, managers.WatchdogTimeout)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "apply failed (rolled back)"))
		return
	}

	types.OK(w, types.ApplyResult{Status: "applied", SnapID: res.SnapID, Warning: res.Warning})
}
