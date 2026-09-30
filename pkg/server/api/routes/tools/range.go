package tools

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/net/ipcalc"
	"github.com/ChevalRouting/routier/pkg/types"
)

// Range godoc
// @Summary  Split an inclusive address range into the minimal set of CIDR blocks
// @Tags tools
// @Produce json
// @Param start query string true "First address in the range"
// @Param end query string true "Last address in the range"
// @Success 200 {object} types.Response[ipcalc.RangeCIDRs]
// @Failure 400 {object} types.Response[any]
// @Security BearerAuth
// @Router /api/tools/range [get]
func Range(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start, end := q.Get("start"), q.Get("end")
	if start == "" || end == "" {
		types.Err(http.StatusBadRequest, "missing start or end parameter").Write(w)
		return
	}

	out, err := ipcalc.Range(start, end)
	if err != nil {
		types.Err(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	types.OK(w, *out)
}
