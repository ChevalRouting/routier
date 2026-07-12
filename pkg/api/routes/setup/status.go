package setup

import (
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
)

// Status godoc
// @Summary  Onboarding / setup status
// @Tags setup
// @Produce json
// @Success 200 {object} types.Response[types.SetupStatus]
// @Security BearerAuth
// @Router /api/setup/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())

	hasIfaces := false
	if cfg, err := config.Load(app.ConfigPath); err == nil {
		hasIfaces = len(cfg.Interfaces) > 0
	}

	types.OK(w, types.SetupStatus{
		NeedsPasswordChange: webdb.Setting(app.DB, webdb.SettingPasswordChanged, "true") != "true",
		OnboardingComplete:  webdb.Setting(app.DB, webdb.SettingOnboardingComplete, "true") == "true",
		HasInterfaces:       hasIfaces,
	})
}
