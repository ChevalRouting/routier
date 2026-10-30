package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/ChevalRouting/routier/pkg/host/updates"
	"github.com/ChevalRouting/routier/pkg/managers"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsStream struct {
	args     func(app *appctx.App) []string
	detached bool
}

var wsRegistry = map[string]wsStream{
	"debug":   {args: debugStreamArgs},
	"reapply": {detached: true, args: reapplyStreamArgs},
	"upgrade": {args: func(*appctx.App) []string {
		return []string{"tail", "-n", "+1", "-F", updates.LogPath}
	}},
}

type wsClientMsg struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// @Summary  Interactive exec/terminal websocket
// @Tags ws
// @Param name query string true "stream name (debug/reapply)"
// @Success 101 {string} string "switching protocols"
// @Security BearerAuth
// @Router /api/ws/exec [get]
func Exec(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	stream, ok := wsRegistry[name]
	if !ok {
		types.Err(http.StatusBadRequest, "unknown stream: "+name).Write(w)
		return
	}

	var osUser *user.User
	if name == "debug" {
		u, err := consoleUser(appctx.UnixUserFromContext(r.Context()))
		if err != nil {
			types.Err(http.StatusForbidden, err.Error()).Write(w)
			return
		}

		osUser = u
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("ws upgrade")
		return
	}

	defer func() { _ = conn.Close() }()

	args := stream.args(appctx.FromContext(r.Context()))
	if stream.detached {
		runDetachedStream(conn, args)
		return
	}

	cmd := exec.CommandContext(r.Context(), args[0], args[1:]...)
	cmd.Env = append(cmd.Environ(), "TERM=xterm-256color")

	if osUser != nil {
		if err := runAsUser(cmd, osUser); err != nil {
			log.Error().Err(err).Str("user", osUser.Username).Msg("debug console: drop privileges")
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"failed to start process"}`))
			return
		}
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Error().Err(err).Strs("args", args).Msg("pty start")
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"failed to start process"}`))
		return
	}

	defer func() { stopProcess(cmd, ptmx) }()

	go copyTerminalOutput(conn, ptmx)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg wsClientMsg
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}

		switch msg.Type {
		case "input":
			_, _ = io.WriteString(ptmx, msg.Data)
		case "resize":
			if msg.Cols > 0 && msg.Rows > 0 {
				_ = pty.Setsize(ptmx, &pty.Winsize{Cols: msg.Cols, Rows: msg.Rows})
			}
		}
	}
}

func consoleUser(username string) (*user.User, error) {
	if username == "" || username == "root" {
		return nil, errors.New("the debug console needs a unix user session")
	}

	osUser, err := user.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("the debug console needs a unix user session: %s has no unix account", username)
	}

	return osUser, nil
}

func runAsUser(cmd *exec.Cmd, osUser *user.User) error {
	uid, err := strconv.ParseUint(osUser.Uid, 10, 32)
	if err != nil {
		return fmt.Errorf("uid %q: %w", osUser.Uid, err)
	}

	gid, err := strconv.ParseUint(osUser.Gid, 10, 32)
	if err != nil {
		return fmt.Errorf("gid %q: %w", osUser.Gid, err)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: supplementaryGroups(osUser)},
	}
	cmd.Env = append(cmd.Env, "HOME="+osUser.HomeDir, "USER="+osUser.Username, "LOGNAME="+osUser.Username)
	if info, err := os.Stat(osUser.HomeDir); err == nil && info.IsDir() {
		cmd.Dir = osUser.HomeDir
	}

	return nil
}

func supplementaryGroups(osUser *user.User) []uint32 {
	ids, err := osUser.GroupIds()
	if err != nil {
		return nil
	}

	groups := make([]uint32, 0, len(ids))
	for _, id := range ids {
		if gid, err := strconv.ParseUint(id, 10, 32); err == nil {
			groups = append(groups, uint32(gid))
		}
	}

	return groups
}

func runDetachedStream(conn *websocket.Conn, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(cmd.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		cancel()
		log.Error().Err(err).Strs("args", args).Msg("pty start")
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"failed to start process"}`))
		return
	}

	go copyTerminalOutput(conn, ptmx)

	go closeWhenExited(conn, cancel, cmd, ptmx)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func stopProcess(cmd *exec.Cmd, ptmx *os.File) {
	_ = ptmx.Close()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	_ = cmd.Wait()
}

func copyTerminalOutput(conn *websocket.Conn, ptmx *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
				return
			}
		}

		if err != nil {
			return
		}
	}
}

func closeWhenExited(conn *websocket.Conn, cancel context.CancelFunc, cmd *exec.Cmd, ptmx *os.File) {
	_ = cmd.Wait()
	_ = ptmx.Close()
	cancel()
	_ = conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))
	_ = conn.Close()
}

func debugStreamArgs(*appctx.App) []string {
	for _, sh := range []string{"/bin/bash", "/bin/ash", "/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return []string{sh, "-i"}
		}
	}

	return []string{"/bin/sh", "-i"}
}

func reapplyStreamArgs(app *appctx.App) []string {
	exe := "routier"
	if p, err := exec.LookPath("routier"); err == nil {
		exe = p
	}

	return []string{exe, "apply", "--source", "web", "--timeout", strconv.Itoa(managers.WatchdogTimeout), app.ConfigPath}
}
