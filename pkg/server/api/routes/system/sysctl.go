package system

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/host/sysctl"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// @Summary Kernel parameters available on this host with their current values
// @Tags system
// @Produce json
// @Success 200 {object} types.Response[[]sysctl.Parameter]
// @Security BearerAuth
// @Router /api/system/sysctl [get]
func Sysctl(w http.ResponseWriter, r *http.Request) {
	params, err := sysctl.List(sysctl.Root)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read kernel parameters"))
		return
	}

	types.OK(w, params)
}
