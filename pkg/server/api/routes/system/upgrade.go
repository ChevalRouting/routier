package system

import (
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/ChevalRouting/routier/pkg/host/updates"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary  Current upgrade job status
// @Tags system
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/system/upgrade [get]
func UpgradeStatus(w http.ResponseWriter, _ *http.Request) {
	types.OK(w, updates.ReadStatus())
}

// @Summary  Start a whole-system upgrade
// @Tags system
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/system/upgrade [post]
func StartUpgrade(w http.ResponseWriter, _ *http.Request) {
	if updates.ReadStatus().State == updates.StateRunning {
		types.OK(w, types.StatusResponse{Status: "running"})
		return
	}

	_ = os.WriteFile(updates.LogPath, nil, 0644)
	_ = updates.WriteStatus(updates.JobStatus{State: updates.StateRunning, StartedAt: time.Now()})

	if err := exec.Command("rc-service", "routier-upgrade", "restart").Run(); err != nil {
		_ = updates.WriteStatus(updates.JobStatus{State: updates.StateFailed, Error: err.Error()})
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to start upgrade"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "started"})
}

// @Summary  Reboot the appliance
// @Tags system
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/system/reboot [post]
func Reboot(w http.ResponseWriter, _ *http.Request) {
	go func() {
		time.Sleep(time.Second)
		_ = exec.Command("reboot").Run()
	}()

	types.OK(w, types.StatusResponse{Status: "rebooting"})
}

// @Summary  Power off the appliance
// @Tags system
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/system/shutdown [post]
func Shutdown(w http.ResponseWriter, _ *http.Request) {
	go func() {
		time.Sleep(time.Second)
		_ = exec.Command("poweroff").Run()
	}()

	types.OK(w, types.StatusResponse{Status: "shutting down"})
}
