package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/ChevalRouting/routier/pkg/bondstat"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type interfaceStatus struct {
	Link map[string]any   `json:"link"`
	Addr string           `json:"addr,omitempty" validate:"optional"`
	Bond *bondstat.Status `json:"bond,omitempty" validate:"optional"`
}

// InterfaceStatus godoc
// @Summary Detailed live kernel interface and bonding state
// @Tags system
// @Produce json
// @Param name path string true "Kernel interface name"
// @Success 200 {object} types.Response[interfaceStatus]
// @Security BearerAuth
// @Router /api/system/interfaces/{name} [get]
func InterfaceStatus(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" || len(name) > 15 || strings.ContainsAny(name, "/\x00 \t\n") || name == "." || name == ".." {
		types.Error(log.Logger, w, types.Err(http.StatusBadRequest, "invalid interface name"))
		return
	}
	if _, err := net.InterfaceByName(name); err != nil {
		types.Error(log.Logger, w, types.Err(http.StatusNotFound, "interface no longer exists"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "ip", args...).Output()
	}
	link, err := readInterfaceLink(ctx, name, run)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to read interface status"))
		return
	}
	addr, err := run(ctx, "-d", "addr", "show", "dev", name)
	if err != nil {
		log.Warn().Err(err).Str("interface", name).Msg("failed to read interface addresses")
	}
	types.OK(w, interfaceStatus{Link: link, Addr: string(addr), Bond: bondstat.Read(name)})
}

// Keep all iproute2 fields so new link types and driver-specific details remain visible.
func readInterfaceLink(ctx context.Context, name string, run func(context.Context, ...string) ([]byte, error)) (map[string]any, error) {
	data, err := run(ctx, "-j", "-d", "-s", "link", "show", "dev", name)
	if err != nil {
		return nil, err
	}
	var links []map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&links); err != nil {
		return nil, err
	}
	if len(links) != 1 || links[0]["ifname"] != name {
		return nil, fmt.Errorf("interface %q missing from link response", name)
	}
	return links[0], nil
}
