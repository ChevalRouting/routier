package install

import (
	"net/http"
	"os/exec"
	"slices"

	"github.com/ChevalRouting/routier/pkg/host/live"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func Routes(r chi.Router) {
	r.Get("/api/install/status", Status)
	r.Get("/api/install/disks", Disks)
	r.Get("/api/install/timezones", Timezones)
	r.Post("/api/install", Start)
	r.Post("/api/install/reboot", Reboot)
}

func requireLive(w http.ResponseWriter) bool {
	if live.Booted(live.MountsPath) {
		return true
	}

	types.Err(http.StatusForbidden, "disk installation is only available on a live system").Write(w)
	return false
}

// @Summary  Live system and disk installation status
// @Tags install
// @Produce json
// @Success 200 {object} types.Response[types.InstallStatus]
// @Security BearerAuth
// @Router /api/install/status [get]
func Status(w http.ResponseWriter, _ *http.Request) {
	status := current.snapshot()
	status.Live = live.Booted(live.MountsPath)
	status.InstallerOnly = status.Live && live.InstallerOnly()
	types.OK(w, status)
}

// @Summary  Disks available for installation
// @Tags install
// @Produce json
// @Success 200 {object} types.Response[[]live.Disk]
// @Security BearerAuth
// @Router /api/install/disks [get]
func Disks(w http.ResponseWriter, _ *http.Request) {
	if !requireLive(w) {
		return
	}

	disks, err := live.Disks()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "list disks"))
		return
	}

	types.OK(w, disks)
}

// @Summary  Timezones available for installation
// @Tags install
// @Produce json
// @Success 200 {object} types.Response[[]string]
// @Security BearerAuth
// @Router /api/install/timezones [get]
func Timezones(w http.ResponseWriter, _ *http.Request) {
	types.OK(w, live.Timezones(live.ZoneinfoDir))
}

// @Summary  Install Routier to a disk, erasing it
// @Tags install
// @Produce json
// @Param body body types.InstallRequest true "target disk and system settings"
// @Success 200 {object} types.Response[types.InstallStatus]
// @Security BearerAuth
// @Router /api/install [post]
func Start(w http.ResponseWriter, r *http.Request) {
	if !requireLive(w) {
		return
	}

	app := appctx.FromContext(r.Context())
	var req types.InstallRequest
	if ve := app.DecodeAndValidate(r, &req); ve != nil {
		ve.Write(w)
		return
	}

	disks, err := live.Disks()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "list disks"))
		return
	}

	if !slices.ContainsFunc(disks, func(d live.Disk) bool { return d.Path == req.Disk && !d.BootMedia }) {
		types.Err(http.StatusUnprocessableEntity, "unknown disk or disk holding the boot media").Write(w)
		return
	}

	if !slices.Contains(live.Timezones(live.ZoneinfoDir), req.Timezone) {
		types.Err(http.StatusUnprocessableEntity, "unknown timezone").Write(w)
		return
	}

	if !current.begin(req.Disk) {
		types.Err(http.StatusConflict, "an installation already ran").Write(w)
		return
	}

	log.Info().Str("disk", req.Disk).Str("hostname", req.Hostname).Msg("disk installation started from the web UI")
	go current.run(req)

	types.OK(w, current.snapshot())
}

// @Summary  Restart the live system after installation
// @Tags install
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/install/reboot [post]
func Reboot(w http.ResponseWriter, _ *http.Request) {
	if !requireLive(w) {
		return
	}

	if current.snapshot().State != "done" {
		types.Err(http.StatusConflict, "no completed installation").Write(w)
		return
	}

	types.OK(w, types.StatusResponse{Status: "rebooting"})
	go func() { _ = exec.Command("reboot").Run() }()
}
