package tools

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/net/ipcalc"
	"github.com/ChevalRouting/routier/pkg/types"
)

// Subnet godoc
// @Summary  Compute an ipcalc-style subnet breakdown for an address or CIDR
// @Tags tools
// @Produce json
// @Param cidr query string true "Address or CIDR, e.g. 100.64.12.192/27 or 2001:db8::/64"
// @Success 200 {object} types.Response[ipcalc.SubnetInfo]
// @Failure 400 {object} types.Response[any]
// @Security BearerAuth
// @Router /api/tools/subnet [get]
func Subnet(w http.ResponseWriter, r *http.Request) {
	cidr := r.URL.Query().Get("cidr")
	if cidr == "" {
		types.Err(http.StatusBadRequest, "missing cidr parameter").Write(w)
		return
	}

	info, err := ipcalc.Subnet(cidr)
	if err != nil {
		types.Err(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	types.OK(w, *info)
}
