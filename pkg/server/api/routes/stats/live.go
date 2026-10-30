package stats

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/server/api/sse"
	"github.com/ChevalRouting/routier/pkg/telemetry/stats"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const liveInterval = time.Second

// @Summary Stream live interface throughput every second (SSE)
// @Tags stats
// @Produce text/event-stream
// @Param iface query string true "Kernel interface name"
// @Success 200 {string} string
// @Security BearerAuth
// @Router /api/stats/live [get]
func Live(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("iface")
	if name == "" || len(name) > 15 || strings.ContainsAny(name, "/\x00 \t\n") || name == "." || name == ".." {
		types.Error(log.Logger, w, types.Err(http.StatusBadRequest, "invalid interface name"))
		return
	}

	if _, err := net.InterfaceByName(name); err != nil {
		types.Error(log.Logger, w, types.Err(http.StatusNotFound, "interface not found"))
		return
	}

	prev, err := stats.ReadIfaceCounters(name)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to read interface counters"))
		return
	}

	stream := sse.Start(w)
	if !stream.Write(": connected\n\n") {
		return
	}

	ticker := time.NewTicker(liveInterval)
	defer ticker.Stop()

	last := time.Now()
	for {
		select {
		case <-r.Context().Done():
			return
		case now := <-ticker.C:
			cur, err := stats.ReadIfaceCounters(name)
			if err != nil {
				return
			}

			sample := stats.NewLiveSample(prev, cur, now.Sub(last), now)
			sample.OperState = stats.ReadOperState(name)
			prev, last = cur, now
			data, err := json.Marshal(sample)
			if err != nil {
				return
			}

			if !stream.Data(string(data)) {
				return
			}
		}
	}
}
