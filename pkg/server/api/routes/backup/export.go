package backup

import (
	"fmt"
	"net/http"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/cfgstore"
	backuppkg "github.com/ChevalRouting/routier/pkg/backup"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Export godoc
// @Summary  Download an encrypted config backup
// @Tags backup
// @Produce octet-stream
// @Success 200 {string} string "encrypted backup archive"
// @Security BearerAuth
// @Router /api/backup/export [get]
func Export(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	cfg, err := cfgstore.Read(app.ConfigPath, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	name := fmt.Sprintf("routier-backup-%s.bin", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	_ = backuppkg.Write(w, app.ConfigPath, cfg)
}
