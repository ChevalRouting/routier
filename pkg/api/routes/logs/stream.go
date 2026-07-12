package logs

import (
	"bufio"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
)

// Stream godoc
// @Summary  Stream system logs (SSE)
// @Tags logs
// @Produce text/event-stream
// @Param source query string true "messages or dmesg"
// @Success 200 {string} string
// @Security BearerAuth
// @Router /api/logs/stream [get]
func Stream(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, hasFlusher := w.(http.Flusher)

	send := func(line string) {
		fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(line, "\n", " "))
		if hasFlusher {
			flusher.Flush()
		}
	}

	switch source {
	case "messages":
		streamLines(exec.CommandContext(r.Context(), "tail", "-n", "100", "-f", "/var/log/messages"), false, send)
	case "dmesg":
		if hist, err := exec.CommandContext(r.Context(), "dmesg").Output(); err == nil {
			for _, line := range strings.Split(strings.TrimRight(string(hist), "\n"), "\n") {
				if line != "" {
					send(line)
				}
			}
		}

		streamLines(exec.CommandContext(r.Context(), "tail", "-n", "0", "-f", "/proc/kmsg"), true, send)
	default:
		types.Err(http.StatusBadRequest, "unknown source").Write(w)
	}
}

func streamLines(cmd *exec.Cmd, stripPriority bool, send func(string)) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	if err := cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if stripPriority {
			if i := strings.IndexByte(line, '>'); i > 0 && line[0] == '<' {
				line = line[i+1:]
			}
		}

		send(line)
	}

	_ = cmd.Wait()
}
