package tools

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/net/ipcalc"
	"github.com/ChevalRouting/routier/pkg/types"
)

// Reverse godoc
// @Summary  Resolve the reverse-DNS (PTR) name for an address, plus the delegation zone for a CIDR
// @Tags tools
// @Produce json
// @Param ip query string true "Address or CIDR, e.g. 100.64.12.192 or 2001:db8::/32"
// @Success 200 {object} types.Response[ipcalc.ReverseDNS]
// @Failure 400 {object} types.Response[any]
// @Security BearerAuth
// @Router /api/tools/reverse [get]
func Reverse(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		types.Err(http.StatusBadRequest, "missing ip parameter").Write(w)
		return
	}

	out, err := ipcalc.Reverse(ip)
	if err != nil {
		types.Err(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	types.OK(w, *out)
}
