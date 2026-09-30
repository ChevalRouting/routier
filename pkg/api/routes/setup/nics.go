package setup

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// Nics godoc
// @Summary  List candidate system NICs
// @Tags setup
// @Produce json
// @Success 200 {object} types.Response[[]types.SystemNic]
// @Security BearerAuth
// @Router /api/system/nics [get]
func Nics(w http.ResponseWriter, _ *http.Request) {
	nics, err := systemNics()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "list interfaces"))
		return
	}

	types.OK(w, nics)
}
