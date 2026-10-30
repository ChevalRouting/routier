package logs

import (
	"bufio"
	"net/http"
	"os/exec"
	"strings"

	"github.com/ChevalRouting/routier/pkg/server/api/sse"
	"github.com/ChevalRouting/routier/pkg/types"
)

// @Summary  Stream system logs (SSE)
// @Tags logs
// @Produce text/event-stream
// @Param source query string true "messages or dmesg"
// @Success 200 {string} string
// @Security BearerAuth
// @Router /api/logs/stream [get]
func Stream(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("source") {
	case "messages":
		streamLines(exec.CommandContext(r.Context(), "tail", "-n", "100", "-f", "/var/log/messages"), false, sse.Start(w))
	case "dmesg":
		stream := sse.Start(w)
		if hist, err := exec.CommandContext(r.Context(), "dmesg").Output(); err == nil {
			for _, line := range strings.Split(strings.TrimRight(string(hist), "\n"), "\n") {
				if line != "" {
					stream.Data(line)
				}
			}
		}

		streamLines(exec.CommandContext(r.Context(), "tail", "-n", "0", "-f", "/proc/kmsg"), true, stream)
	default:
		types.Err(http.StatusBadRequest, "unknown source").Write(w)
	}
}

func streamLines(cmd *exec.Cmd, stripPriority bool, stream *sse.Stream) {
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

		if !stream.Data(line) {
			break
		}
	}

	_ = cmd.Wait()
}
