package flows

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/telemetry/flows"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const (
	defaultMinutes  = 60
	maxMinutes      = 7 * 24 * 60
	defaultListSize = 200
	maxListSize     = 1000
	defaultTopSize  = 10
	maxTopSize      = 100
	seriesMaxPoints = 120
	defaultPaths    = 15
	maxPaths        = 50
	maxPathDepth    = 3
)

var defaultPathDimensions = []string{"src", "service", "dst"}

var dimensions = map[string]bool{"src": true, "dst": true, "service": true, "protocol": true, "in_if": true, "out_if": true, "conversation": true}

func boundedQuery(r *http.Request, name string, def, maxValue int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil || v <= 0 {
		return def
	}

	return min(v, maxValue)
}

func pathDimensions(raw string) ([]string, error) {
	if raw == "" {
		return defaultPathDimensions, nil
	}

	parts := strings.Split(raw, ",")
	if len(parts) < 2 || len(parts) > maxPathDepth {
		return nil, errors.New("dimensions takes two or three values")
	}

	for i, part := range parts {
		if !dimensions[part] || part == "conversation" {
			return nil, errors.New("unknown dimension: " + part)
		}

		if slices.Contains(parts[:i], part) {
			return nil, errors.New("repeated dimension: " + part)
		}
	}

	return parts, nil
}

func since(r *http.Request) int64 {
	return time.Now().Unix() - int64(boundedQuery(r, "minutes", defaultMinutes, maxMinutes))*60
}

// @Summary  Flow collector status
// @Description Reports whether the sFlow agent runs, its sampling rate, interfaces, collectors and how many flows are stored locally.
// @Tags flows
// @Produce json
// @Success 200 {object} types.Response[types.FlowStatus]
// @Security BearerAuth
// @Router /api/flows/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "load config"))
		return
	}

	status := flows.Status(cfg.Monitoring.FlowsEnabled())
	if stored, err := webdb.CountFlows(r.Context(), app.DB); err == nil {
		status.Stored = stored
	}

	types.OK(w, status)
}

// @Summary  Recent flows
// @Tags flows
// @Produce json
// @Param minutes query int false "lookback minutes (default 60)"
// @Param address query string false "only flows from or to this address"
// @Param limit query int false "maximum flows (default 200)"
// @Success 200 {object} types.Response[types.FlowsResponse]
// @Security BearerAuth
// @Router /api/flows [get]
func List(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	rows, err := webdb.RecentFlows(r.Context(), app.DB, since(r), r.URL.Query().Get("address"), boundedQuery(r, "limit", defaultListSize, maxListSize))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "query flows"))
		return
	}

	types.OK(w, types.FlowsResponse{Flows: rows})
}

// @Summary  Top flow talkers
// @Description Sums stored flows by one dimension: src, dst, service (protocol/port), protocol, in_if, out_if or conversation.
// @Tags flows
// @Produce json
// @Param dimension query string false "grouping dimension (default src)"
// @Param minutes query int false "lookback minutes (default 60)"
// @Param limit query int false "maximum entries (default 10)"
// @Success 200 {object} types.Response[types.FlowTopResponse]
// @Security BearerAuth
// @Router /api/flows/top [get]
func Top(w http.ResponseWriter, r *http.Request) {
	dimension := r.URL.Query().Get("dimension")
	if dimension == "" {
		dimension = "src"
	}

	if !dimensions[dimension] {
		types.Err(http.StatusBadRequest, "unknown dimension: "+dimension).Write(w)
		return
	}

	app := appctx.FromContext(r.Context())
	entries, err := webdb.TopFlows(r.Context(), app.DB, dimension, since(r), boundedQuery(r, "limit", defaultTopSize, maxTopSize))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "query top flows"))
		return
	}

	types.OK(w, types.FlowTopResponse{Dimension: dimension, Entries: entries})
}

// @Summary  Flow paths for a Sankey diagram
// @Description Sums stored flows along two or three dimensions (src, dst, service, protocol, in_if, out_if) and returns the largest paths with the total bytes of the window. Names maps path addresses to their cached reverse DNS (PTR) names.
// @Tags flows
// @Produce json
// @Param dimensions query string false "comma-separated dimensions (default src,service,dst)"
// @Param minutes query int false "lookback minutes (default 60)"
// @Param limit query int false "maximum paths (default 15)"
// @Success 200 {object} types.Response[types.FlowSankeyResponse]
// @Security BearerAuth
// @Router /api/flows/sankey [get]
func Sankey(w http.ResponseWriter, r *http.Request) {
	dims, err := pathDimensions(r.URL.Query().Get("dimensions"))
	if err != nil {
		types.Err(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	app := appctx.FromContext(r.Context())
	from := since(r)
	paths, err := webdb.FlowPaths(r.Context(), app.DB, dims, from, boundedQuery(r, "limit", defaultPaths, maxPaths))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "query flow paths"))
		return
	}

	total, err := webdb.FlowBytesSince(r.Context(), app.DB, from)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "sum flow bytes"))
		return
	}

	names, err := webdb.FlowNames(r.Context(), app.DB, pathAddresses(dims, paths))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "query flow names"))
		return
	}

	types.OK(w, types.FlowSankeyResponse{Dimensions: dims, Total: total, Paths: paths, Names: names})
}

func pathAddresses(dims []string, paths []types.FlowPath) []string {
	var addresses []string
	for _, p := range paths {
		for i, key := range p.Keys {
			if (dims[i] == "src" || dims[i] == "dst") && key != "" && !slices.Contains(addresses, key) {
				addresses = append(addresses, key)
			}
		}
	}

	return addresses
}

// @Summary  Flow traffic over time
// @Description Bytes and packets per input interface, bucketed over the lookback window.
// @Tags flows
// @Produce json
// @Param minutes query int false "lookback minutes (default 60)"
// @Success 200 {object} types.Response[types.FlowSeriesResponse]
// @Security BearerAuth
// @Router /api/flows/series [get]
func Series(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	window := int64(boundedQuery(r, "minutes", defaultMinutes, maxMinutes)) * 60
	bucket := webdb.HistoryBucket(window, seriesMaxPoints)
	points, err := webdb.FlowSeries(r.Context(), app.DB, time.Now().Unix()-window, bucket)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "query flow series"))
		return
	}

	types.OK(w, types.FlowSeriesResponse{Bucket: bucket, Points: points})
}

// @Summary  Interface metadata for akvorado
// @Description Lists this router as an sFlow exporter with its interface indexes, names and speeds, in the format of akvorado's static provider exporter-sources. Use the transform .result.exporters[].
// @Tags flows
// @Produce json
// @Success 200 {object} types.Response[flows.Metadata]
// @Security BearerAuth
// @Router /api/flows/metadata [get]
func Metadata(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	cfg, err := config.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "load config"))
		return
	}

	config.ResolveInterfaces(cfg)
	settings, _ := flows.SettingsFromConfig(cfg)
	types.OK(w, flows.BuildMetadata(cfg, flows.ExporterAddresses(settings), flows.SystemInterfaces(), flows.LinkSpeed))
}
