package routing

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	"github.com/ChevalRouting/routier/pkg/api/routecache"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

// KernelRoutes godoc
// @Summary  Kernel routing table (filtered, paged)
// @Tags routing
// @Produce json
// @Param q query string false "text filter"
// @Param family query string false "ip family"
// @Param proto query string false "protocols, comma separated"
// @Param offset query int false "page offset"
// @Param limit query int false "page size"
// @Param default_only query bool false "only default routes"
// @Success 200 {object} types.Response[types.RoutesResponse]
// @Security BearerAuth
// @Router /api/routing/routes [get]
func KernelRoutes(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	query := r.URL.Query()

	var protoKeywords []string
	for p := range strings.SplitSeq(query.Get("proto"), ",") {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			protoKeywords = append(protoKeywords, p)
		}
	}

	filter := webdb.Filter{
		Query:       strings.ToLower(query.Get("q")),
		Family:      query.Get("family"),
		Protocols:   protoKeywords,
		DefaultOnly: query.Get("default_only") == "true" || query.Get("default_only") == "1",
	}

	offset := parseQueryInt(query.Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}

	limit := parseQueryInt(query.Get("limit"), 200)
	if limit <= 0 {
		limit = 200
	}

	var routesTTL time.Duration
	if cfg, err := cfgstore.Read(app.ConfigPath, ""); err == nil && cfg.Monitoring != nil && cfg.Monitoring.Collection.Routes > 0 {
		routesTTL = time.Duration(cfg.Monitoring.Collection.Routes) * time.Second
	}

	force := query.Get("refresh") == "1" || query.Get("refresh") == "true"
	if err := routecache.MaybeRefresh(app.DB, force, routesTTL); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "refresh route snapshot"))
		return
	}

	routes, total, err := webdb.QueryKernelRoutes(app.DB, filter, offset, limit)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "query routes"))
		return
	}

	types.OK(w, types.RoutesResponse{Routes: routes, Total: total})
}

func parseQueryInt(s string, def int) int {
	if s == "" {
		return def
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}

	return n
}
