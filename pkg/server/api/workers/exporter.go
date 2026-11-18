package workers

import (
	"context"
	"errors"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/telemetry/exporter"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const (
	exporterConfigInterval = 10 * time.Second
	exporterProbeWindow    = time.Hour
)

type exporterRunner struct {
	db         *webdb.DB
	configPath string
	servers    map[string]*http.Server
	wg         sync.WaitGroup
}

func StartExporter(ctx context.Context, g *Group, db *webdb.DB, configPath string) {
	r := &exporterRunner{db: db, configPath: configPath, servers: map[string]*http.Server{}}
	g.Go(func() { r.run(ctx) })
}

func (r *exporterRunner) run(ctx context.Context) {
	ticker := time.NewTicker(exporterConfigInterval)
	defer ticker.Stop()

	for {
		r.reconcile()
		select {
		case <-ctx.Done():
			r.stop()
			return
		case <-ticker.C:
		}
	}
}

func (r *exporterRunner) reconcile() {
	cfg, err := config.Load(r.configPath)
	if err != nil {
		return
	}

	var addresses []string
	if cfg.Monitoring.ExporterEnabled() {
		addresses = cfg.Monitoring.Exporter.Addresses()
	}

	for address, server := range r.servers {
		if !slices.Contains(addresses, address) {
			_ = server.Close()
			delete(r.servers, address)
		}
	}

	for _, address := range addresses {
		if _, ok := r.servers[address]; !ok {
			r.listen(address)
		}
	}
}

func (r *exporterRunner) listen(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Error().Err(err).Str("address", address).Msg("exporter: listen failed")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", r.serveMetrics)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	r.servers[address] = server
	r.wg.Add(1)
	go r.serve(server, listener, address)
}

func (r *exporterRunner) serve(server *http.Server, listener net.Listener, address string) {
	defer r.wg.Done()
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error().Err(err).Str("address", address).Msg("exporter: server stopped")
	}
}

func (r *exporterRunner) stop() {
	for _, server := range r.servers {
		_ = server.Close()
	}

	r.wg.Wait()
	clear(r.servers)
}

func (r *exporterRunner) serveMetrics(w http.ResponseWriter, req *http.Request) {
	snapshot := exporter.Snapshot{Version: app.Version}
	live := LiveSnapshot()
	snapshot.System, snapshot.BGP, snapshot.OSPF = live.Sys, live.BGP, live.OSPF
	if cfg, err := config.Load(r.configPath); err == nil {
		config.ResolveInterfaces(cfg)
		snapshot.Interfaces = interfaceNames(cfg)
		snapshot.Probes = r.latestProbes(req.Context(), cfg)
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = exporter.Collect(snapshot).WriteTo(w)
}

func interfaceNames(cfg *config.Config) map[string]string {
	out := map[string]string{}
	for name, iface := range cfg.Interfaces {
		if iface.Device != "" {
			out[iface.Device] = name
		}
	}

	return out
}

func (r *exporterRunner) latestProbes(ctx context.Context, cfg *config.Config) []types.ProbeHistoryPoint {
	if cfg.Monitoring == nil || len(cfg.Monitoring.Probes) == 0 {
		return nil
	}

	history := webdb.ProbeHistory(ctx, r.db, time.Now().Add(-exporterProbeWindow).Unix(), 1)
	var out []types.ProbeHistoryPoint
	for _, probe := range cfg.Monitoring.Probes {
		points := history[probe.Name]
		if len(points) > 0 {
			out = append(out, points[len(points)-1])
		}
	}

	return out
}
