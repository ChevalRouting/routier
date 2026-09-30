package system

import (
	"github.com/ChevalRouting/routier/pkg/telemetry/bondstat"
	"github.com/ChevalRouting/routier/pkg/types"
	"net/http"
)

// Bonds godoc
// @Summary Live bond link and LACP state
// @Tags system
// @Produce json
// @Success 200 {object} types.Response[[]bondstat.Status]
// @Security BearerAuth
// @Router /api/system/bonds [get]
func Bonds(w http.ResponseWriter, r *http.Request) {
	types.OK(w, bondstat.List())
}
