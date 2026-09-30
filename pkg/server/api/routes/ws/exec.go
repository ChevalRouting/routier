package ws

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/ChevalRouting/routier/pkg/host/updates"

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
	"debug": {args: func(*appctx.App) []string {
		for _, sh := range []string{"/bin/bash", "/bin/ash", "/bin/sh"} {
			if _, err := os.Stat(sh); err == nil {
				return []string{sh, "-i"}
			}
		}

		return []string{"/bin/sh", "-i"}
	}},
	"reapply": {detached: true, args: func(app *appctx.App) []string {
		exe := "routier"
		if p, err := exec.LookPath("routier"); err == nil {
			exe = p
		}

		return []string{exe, "apply", "--source", "web", "--timeout", strconv.Itoa(managers.WatchdogTimeout), app.ConfigPath}
	}},
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

// Exec godoc
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

	if name == "debug" {
		dropToUnixUser(cmd, appctx.UsernameFromContext(r.Context()))
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Error().Err(err).Strs("args", args).Msg("pty start")
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"failed to start process"}`))
		return
	}

	defer func() {
		ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}

		_ = cmd.Wait()
	}()

	go func() {
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
	}()

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

func dropToUnixUser(cmd *exec.Cmd, username string) {
	if username == "" || username == "root" {
		return
	}

	osUser, err := user.Lookup(username)
	if err != nil {
		log.Warn().Err(err).Str("user", username).Msg("debug console: unix user lookup failed")
		return
	}

	uid, uerr := strconv.Atoi(osUser.Uid)
	gid, gerr := strconv.Atoi(osUser.Gid)
	if uerr != nil || gerr != nil {
		return
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: supplementaryGroups(osUser)},
	}
	cmd.Env = append(cmd.Env, "HOME="+osUser.HomeDir, "USER="+username, "LOGNAME="+username)
	if info, err := os.Stat(osUser.HomeDir); err == nil && info.IsDir() {
		cmd.Dir = osUser.HomeDir
	}
}

func supplementaryGroups(osUser *user.User) []uint32 {
	ids, err := osUser.GroupIds()
	if err != nil {
		return nil
	}

	var groups []uint32
	for _, id := range ids {
		if gid, err := strconv.Atoi(id); err == nil {
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

	go func() {
		buf := make([]byte, 4096)
		for {
			n, rerr := ptmx.Read(buf)
			if n > 0 {
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}

			if rerr != nil {
				return
			}
		}
	}()

	go func() {
		_ = cmd.Wait()
		ptmx.Close()
		cancel()
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second))
		_ = conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
