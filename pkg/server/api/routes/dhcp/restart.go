package dhcp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Restart godoc
// @Summary  Restart the running DHCP servers (Kea)
// @Tags dhcp
// @Produce json
// @Success 200 {object} types.Response[string]
// @Security BearerAuth
// @Router /api/dhcp/restart [post]
func Restart(w http.ResponseWriter, r *http.Request) {
	var restarted []string
	for _, name := range []string{"kea-dhcp4", "kea-dhcp6", "kea-dhcp-ddns"} {
		if !svc.ServiceRunning(name) {
			continue
		}

		if err := svc.RestartService(name); err != nil {
			types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, fmt.Sprintf("failed to restart %s", name)))
			return
		}

		restarted = append(restarted, name)
	}

	if len(restarted) == 0 {
		types.Error(log.Logger, w, types.NewError(http.StatusBadRequest, "no DHCP server is running"))
		return
	}

	types.OK(w, "restarted "+strings.Join(restarted, ", "))
}
