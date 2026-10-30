package wireguard

import (
	"context"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

const (
	wgTimeout       = 5 * time.Second
	interfaceFields = 5
	peerFields      = 9
	noValue         = "(none)"
)

// @Summary  WireGuard interfaces and peers with their latest handshake and traffic
// @Tags wireguard
// @Produce json
// @Success 200 {object} types.Response[[]types.WireguardInterfaceStatus]
// @Security BearerAuth
// @Router /api/wireguard/status [get]
func Status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), wgTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "wg", "show", "all", "dump").Output()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusBadGateway, err, "failed to read WireGuard status"))
		return
	}

	status := ParseDump(string(out))
	if cfg, err := cfgpkg.Load(appctx.FromContext(r.Context()).ConfigPath); err == nil {
		namePeers(status, cfg)
	}

	types.OK(w, status)
}

func ParseDump(dump string) []types.WireguardInterfaceStatus {
	out := []types.WireguardInterfaceStatus{}
	index := map[string]int{}
	for _, line := range strings.Split(strings.TrimSpace(dump), "\n") {
		fields := strings.Split(line, "\t")
		switch len(fields) {
		case interfaceFields:
			port, _ := strconv.Atoi(fields[3])
			index[fields[0]] = len(out)
			out = append(out, types.WireguardInterfaceStatus{Interface: fields[0], PublicKey: fields[2], ListenPort: port, Peers: []types.WireguardPeerStatus{}})
		case peerFields:
			position, ok := index[fields[0]]
			if !ok {
				continue
			}

			out[position].Peers = append(out[position].Peers, parsePeer(fields))
		}
	}

	return out
}

func parsePeer(fields []string) types.WireguardPeerStatus {
	handshake, _ := strconv.ParseInt(fields[5], 10, 64)
	rx, _ := strconv.ParseInt(fields[6], 10, 64)
	tx, _ := strconv.ParseInt(fields[7], 10, 64)
	peer := types.WireguardPeerStatus{PublicKey: fields[1], AllowedIPs: []string{}, LatestHandshake: handshake, RxBytes: rx, TxBytes: tx}
	if fields[3] != noValue {
		peer.Endpoint = fields[3]
	}

	if fields[4] != noValue && fields[4] != "" {
		peer.AllowedIPs = strings.Split(fields[4], ",")
	}

	return peer
}

func namePeers(status []types.WireguardInterfaceStatus, cfg *cfgpkg.Config) {
	for i := range status {
		wg := cfg.VPN.Wireguard[status[i].Interface]
		if wg == nil {
			continue
		}

		for j := range status[i].Peers {
			for _, peer := range wg.Peers {
				if peer.PublicKey == status[i].Peers[j].PublicKey {
					status[i].Peers[j].Name = peer.Name
				}
			}
		}
	}
}
