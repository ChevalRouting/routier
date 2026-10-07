package dns

import io "io"

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/ChevalRouting/routier/pkg/daemon/bind"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

type serviceState struct {
	Service string `json:"service"`
	Running bool   `json:"running"`
}

type statsResponse struct {
	Services []serviceState `json:"services,omitempty" validate:"optional"`
	Status   *bind.Status   `json:"status,omitempty"   validate:"optional"`
	Stats    []bind.Stat    `json:"stats,omitempty"    validate:"optional"`
}

// @Summary  Resolver state and BIND statistics
// @Tags dns
// @Produce json
// @Success 200 {object} types.Response[dns.statsResponse]
// @Security BearerAuth
// @Router /api/dns/stats [get]
func Stats(w http.ResponseWriter, r *http.Request) {
	resp := statsResponse{
		Services: []serviceState{{Service: "named", Running: svc.ServiceRunning("named")}},
	}

	if client, err := bind.LoadLocal(); err == nil {
		if status, err := client.Status(); err == nil {
			resp.Status = status
		}

		if stats, err := client.Statistics(); err == nil {
			resp.Stats = bind.KeyStatistics(stats)
		}
	}

	types.OK(w, resp)
}

// @Summary  Tail the named query log events (SSE)
// @Tags dns
// @Produce text/event-stream
// @Success 200 {string} string
// @Security BearerAuth
// @Router /api/dns/queries/stream [get]
func QueryStream(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	count := "50"
	if r.URL.Query().Get("history") == "0" {
		count = "0"
	}

	cmd := exec.CommandContext(ctx, "tail", "-n", count, "-F", render.NamedLog)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to tail the named log"))
		return
	}

	if err := cmd.Start(); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to start tail"))
		return
	}

	defer func() { cancel(); _ = cmd.Wait() }()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	send := func(frame string) bool { return queryStreamCallback(w, flusher, frame) }
	if !send(": connected\n\n") {
		return
	}

	lines := make(chan string)
	go func() { queryStreamCallback2(ctx, stdout, lines) }()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if !send(": heartbeat\n\n") {
				return
			}
		case line, ok := <-lines:
			if !ok {
				return
			}

			query, ok := bind.ParseQueryLog(line)
			if !ok {
				continue
			}

			payload, err := json.Marshal(query)
			if err == nil && !send(fmt.Sprintf("data: %s\n\n", payload)) {
				return
			}
		}
	}
}

func queryStreamCallback(w http.ResponseWriter, flusher http.Flusher, frame string) bool {
	if _, err := fmt.Fprint(w, frame); err != nil {
		return false
	}

	if flusher != nil {
		flusher.Flush()
	}

	return true
}

func queryStreamCallback2(ctx context.Context, stdout io.ReadCloser, lines chan string) {
	defer close(lines)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case lines <- scanner.Text():
		case <-ctx.Done():
			return
		}
	}
}
