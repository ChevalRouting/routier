package setup

import (
	"net/http"
	"os"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Complete godoc
// @Summary  Mark onboarding complete
// @Tags setup
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/setup/complete [post]
func Complete(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	if err := ensureConfig(app.ConfigPath); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to write config"))
		return
	}

	if err := webdb.SetSetting(requests.DurableContext(r), app.DB, webdb.SettingOnboardingComplete, "true"); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to save setting"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}

func ensureConfig(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "routier"
	}

	return config.Save(path, &config.Config{Version: config.CurrentVersion, Hostname: hostname})
}
