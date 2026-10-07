package system

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/host/boot"
	"github.com/ChevalRouting/routier/pkg/host/updates"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
)

type VersionResponse struct {
	Version       string `json:"version"`
	Kernel        string `json:"kernel"`
	Flavor        string `json:"flavor"`
	RebootPending bool   `json:"reboot_pending"`
	Live          bool   `json:"live"`
}

// @Summary  Current software and kernel version
// @Tags system
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/system/version [get]
func Version(w http.ResponseWriter, _ *http.Request) {
	types.OK(w, VersionResponse{
		Version:       currentVersion(),
		Kernel:        boot.RunningRelease(),
		Flavor:        boot.RunningFlavor(),
		RebootPending: updates.RebootPending(),
		Live:          boot.IsLive(),
	})
}

func currentVersion() string {
	if appctx.Version != "" && appctx.Version != "dev" {
		return appctx.Version
	}

	if v := updates.InstalledVersion(); v != "" {
		return v
	}

	return appctx.Version
}
