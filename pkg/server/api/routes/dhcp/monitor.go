package dhcp

import (
	"bufio"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/ChevalRouting/routier/pkg/daemon/kea"
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
	Stats4   []kea.Stat     `json:"stats4,omitempty" validate:"optional"`
	Stats6   []kea.Stat     `json:"stats6,omitempty" validate:"optional"`
}

// @Summary  DHCP server state and Kea statistics
// @Tags dhcp
// @Produce json
// @Success 200 {object} types.Response[dhcp.statsResponse]
// @Security BearerAuth
// @Router /api/dhcp/stats [get]
func Stats(w http.ResponseWriter, r *http.Request) {
	resp := statsResponse{
		Services: []serviceState{
			{Service: "kea-dhcp4", Running: svc.ServiceRunning("kea-dhcp4")},
			{Service: "kea-dhcp6", Running: svc.ServiceRunning("kea-dhcp6")},
			{Service: "kea-dhcp-ddns", Running: svc.ServiceRunning("kea-dhcp-ddns")},
		},
	}

	if client, err := clientFor(r); err == nil {
		if s, err := client.Statistics("dhcp4"); err == nil {
			resp.Stats4 = s
		}

		if s, err := client.Statistics("dhcp6"); err == nil {
			resp.Stats6 = s
		}
	}

	types.OK(w, resp)
}

// @Summary  Tail Kea DHCP daemon logs (SSE)
// @Tags dhcp
// @Produce text/event-stream
// @Success 200 {string} string
// @Security BearerAuth
// @Param source query string false "Kea daemon (omitted streams DHCPv4 and DHCPv6)" Enums(kea-dhcp4, kea-dhcp6, kea-dhcp-ddns)
// @Router /api/dhcp/leases/stream [get]
func LeaseStream(w http.ResponseWriter, r *http.Request) {
	files, ok := leaseLogFiles(r.URL.Query().Get("source"))
	if !ok {
		types.Err(http.StatusBadRequest, "invalid DHCP log source").Write(w)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, hasFlusher := w.(http.Flusher)
	send := func(line string) { leaseStreamCallback(w, flusher, hasFlusher, line) }

	args := append([]string{"-n", "50", "-F"}, files...)
	cmd := exec.CommandContext(r.Context(), "tail", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to tail kea logs"))
		return
	}

	if err := cmd.Start(); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to start tail"))
		return
	}

	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	if hasFlusher {
		_, _ = fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "==> ") && strings.HasSuffix(line, " <==") {
			continue
		}

		send(line)
	}
}

func leaseStreamCallback(w http.ResponseWriter, flusher http.Flusher, hasFlusher bool, line string) {
	_, _ = fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(line, "\n", " "))
	if hasFlusher {
		flusher.Flush()
	}
}
