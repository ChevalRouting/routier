package setup

import (
	"net/http"
	"os"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
)

const seedPasswordFile = "/var/lib/routier/ui-seed-password"

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

	needsPasswordChange := false
	if _, err := os.Stat(seedPasswordFile); err == nil {
		needsPasswordChange = true
	}

	types.OK(w, types.SetupStatus{
		NeedsPasswordChange: needsPasswordChange,
		OnboardingComplete:  webdb.Setting(r.Context(), app.DB, webdb.SettingOnboardingComplete, "false") == "true",
		HasInterfaces:       hasIfaces,
	})
}
