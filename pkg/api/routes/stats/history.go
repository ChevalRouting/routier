package stats

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	sysstats "github.com/ChevalRouting/routier/pkg/stats"
	"github.com/ChevalRouting/routier/pkg/types"
)

const historyMaxPoints = 200

// History godoc
// @Summary  Historical stats
// @Tags stats
// @Produce json
// @Param minutes query int false "lookback minutes"
// @Param iface query string false "interface filter"
// @Param series query string false "comma-separated series to include (interfaces,bgp,proto,system,total,usage); default all"
// @Success 200 {object} types.Response[types.StatsHistoryResponse]
// @Security BearerAuth
// @Router /api/stats/history [get]
func History(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	minutes := 60
	if m := r.URL.Query().Get("minutes"); m != "" {
		if v, _ := strconv.Atoi(m); v > 0 && v <= 43200 {
			minutes = v
		}
	}

	want := parseSeries(r.URL.Query().Get("series"))
	ifaceFilter := r.URL.Query().Get("iface")
	window := int64(minutes) * 60
	cutoff := time.Now().Unix() - window
	bucket := webdb.HistoryBucket(window, historyMaxPoints)

	resp := types.StatsHistoryResponse{}
	physical := sysstats.PhysicalIfaces()

	var ifaceHistory map[string][]types.IfaceHistoryPoint
	if want("interfaces") || want("usage") {
		ifaceHistory = webdb.IfaceHistory(app.DB, cutoff, bucket, ifaceFilter)
	}

	if want("interfaces") {
		resp.Interfaces = ifaceHistory
	}

	if want("bgp") {
		resp.BGPPeers = webdb.BGPHistory(app.DB, cutoff, bucket)
	}

	if want("proto") {
		resp.Proto = webdb.ProtoHistory(app.DB, cutoff, bucket)
	}

	if want("system") {
		resp.System = webdb.SystemHistory(app.DB, cutoff, bucket)
	}

	if want("total") {
		resp.Total = webdb.IfaceTotals(app.DB, cutoff, bucket, physical)
	}

	if want("usage") {
		resp.Usage = summarizeUsage(ifaceHistory, physical)
	}

	types.OK(w, resp)
}

func parseSeries(raw string) func(string) bool {
	if strings.TrimSpace(raw) == "" {
		return func(string) bool { return true }
	}

	set := map[string]bool{}
	for _, name := range strings.Split(raw, ",") {
		if name = strings.TrimSpace(name); name != "" {
			set[name] = true
		}
	}

	return func(name string) bool { return set[name] }
}

func summarizeUsage(ifaceHistory map[string][]types.IfaceHistoryPoint, physical map[string]bool) *types.UsageSummary {
	usage := &types.UsageSummary{PerIface: make(map[string]types.IfaceUsagePoint)}
	for iface, points := range ifaceHistory {
		var rx, tx float64
		for i := 1; i < len(points); i++ {
			dt := float64(points[i].TS - points[i-1].TS)
			if points[i].RxBytesPS != nil {
				rx += *points[i].RxBytesPS * dt
			}

			if points[i].TxBytesPS != nil {
				tx += *points[i].TxBytesPS * dt
			}
		}

		u := types.IfaceUsagePoint{RxBytes: int64(rx), TxBytes: int64(tx)}
		usage.PerIface[iface] = u
		if len(physical) == 0 || physical[iface] {
			usage.Total.RxBytes += u.RxBytes
			usage.Total.TxBytes += u.TxBytes
		}
	}

	return usage
}
