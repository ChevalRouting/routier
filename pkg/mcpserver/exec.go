package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultExecTimeout = 30 * time.Second
	maxExecTimeout     = 10 * time.Minute
	maxExecOutput      = 1 << 20
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07]*(\x07|\x1b\\)|\r`)

type execResult struct {
	Command  string `json:"command"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

func (i *instance) debugExec(ctx context.Context, command string, timeout time.Duration) (execResult, error) {
	if strings.TrimSpace(command) == "" {
		return execResult{}, fmt.Errorf("command is required")
	}

	if timeout <= 0 {
		timeout = defaultExecTimeout
	}

	if timeout > maxExecTimeout {
		timeout = maxExecTimeout
	}

	endpoint, err := i.wsExecURL()
	if err != nil {
		return execResult{}, err
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: i.insecure},
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, resp, err := dialer.DialContext(dialCtx, endpoint, http.Header{"Authorization": {"Bearer " + i.token}})
	if err != nil {
		if resp != nil {
			return execResult{}, fmt.Errorf("%s: debug console handshake failed: HTTP %d", i.name, resp.StatusCode)
		}

		return execResult{}, fmt.Errorf("%s: debug console dial: %w", i.name, err)
	}

	defer conn.Close()

	marker := execMarker()
	start := marker + "START"
	end := marker + "END"

	script := fmt.Sprintf("printf '%%sSTART\\n' '%s'; %s; printf '%%sEND %%d\\n' '%s' \"$?\"; exit\n", marker, command, marker)
	if err := conn.WriteJSON(map[string]string{"type": "input", "data": script}); err != nil {
		return execResult{}, fmt.Errorf("%s: send command: %w", i.name, err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(timeout))

	var buf strings.Builder
	for {
		mt, data, readErr := conn.ReadMessage()
		if mt == websocket.TextMessage && strings.Contains(string(data), `"type":"error"`) {
			return execResult{}, fmt.Errorf("%s: debug console error: %s", i.name, string(data))
		}

		if len(data) > 0 {
			buf.Write(data)
			if buf.Len() > maxExecOutput || strings.Contains(buf.String(), end) {
				break
			}
		}

		if readErr != nil {
			break
		}
	}

	return parseExecOutput(command, buf.String(), start, end)
}

func parseExecOutput(command, raw, start, end string) (execResult, error) {
	clean := ansiPattern.ReplaceAllString(raw, "")

	startIdx := strings.Index(clean, start+"\n")
	if startIdx < 0 {
		return execResult{}, fmt.Errorf("command did not run to completion (no start marker); raw console output:\n%s", strings.TrimSpace(clean))
	}

	body := clean[startIdx+len(start)+1:]
	endIdx := strings.Index(body, end)
	if endIdx < 0 {
		return execResult{Command: command, Output: strings.TrimRight(body, "\n"), ExitCode: -1}, nil
	}

	output := body[:endIdx]
	code := -1
	if _, err := fmt.Sscanf(body[endIdx:], end+" %d", &code); err != nil {
		code = -1
	}

	return execResult{Command: command, Output: strings.Trim(output, "\n"), ExitCode: code}, nil
}

func (i *instance) wsExecURL() (string, error) {
	switch {
	case strings.HasPrefix(i.url, "https://"):
		return "wss://" + strings.TrimPrefix(i.url, "https://") + "/api/ws/exec?name=debug", nil
	case strings.HasPrefix(i.url, "http://"):
		return "ws://" + strings.TrimPrefix(i.url, "http://") + "/api/ws/exec?name=debug", nil
	default:
		return "", fmt.Errorf("%s: unsupported instance url scheme: %s", i.name, i.url)
	}
}

func execMarker() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "RTR" + hex.EncodeToString(b[:])
}
