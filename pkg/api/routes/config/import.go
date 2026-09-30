package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	friendspkg "github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/rs/zerolog/log"
)

var allowedImportPrefixes = []string{
	"/etc/routier/templates/",
	"/etc/routier/out/",
}

func validateImportFilePath(path string) error {
	clean := filepath.Clean(path)
	for _, prefix := range allowedImportPrefixes {
		if strings.HasPrefix(clean, prefix) {
			return nil
		}
	}

	return fmt.Errorf("path must be under one of: %s", strings.Join(allowedImportPrefixes, ", "))
}

// Import godoc
// @Summary  Import a friend-pushed config payload
// @Tags config
// @Accept json
// @Produce json
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/config/import [put]
func Import(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	username := appctx.UsernameFromContext(r.Context())

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		types.Err(http.StatusBadRequest, "failed to read body").Write(w)
		return
	}

	if name, isFriend := appctx.FriendUsername(r.Context()); isFriend {
		if r.Header.Get("Content-Type") != types.SealedContentType {
			types.Err(http.StatusBadRequest, "friend config push must be encrypted").Write(w)
			return
		}

		if app.Identity == nil {
			types.Err(http.StatusInternalServerError, "no local identity for decryption").Write(w)
			return
		}

		cfg, cerr := cfgpkg.Load(app.ConfigPath)
		if cerr != nil {
			types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, cerr, "read local config"))
			return
		}

		sender := friendspkg.Get(cfg, name)
		if sender == nil || sender.Identity.PublicKey == "" {
			types.Err(http.StatusForbidden, "unknown or unpaired friend").Write(w)
			return
		}

		plain, oerr := app.Identity.Open(body, sender.Identity.PublicKey)
		if oerr != nil {
			types.Error(log.Logger, w, types.Wrap(http.StatusBadRequest, oerr, "decrypt friend payload"))
			return
		}

		body = plain
	}

	var payload managers.PushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		types.Err(http.StatusBadRequest, "invalid payload: "+err.Error()).Write(w)
		return
	}

	if payload.Config == nil {
		types.Err(http.StatusBadRequest, "missing config in payload").Write(w)
		return
	}

	incoming := payload.Config

	for path, content := range payload.Files {
		if err := validateImportFilePath(path); err != nil {
			types.Err(http.StatusBadRequest, fmt.Sprintf("rejected file path %q: %v", path, err)).Write(w)
			return
		}

		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			types.Err(http.StatusInternalServerError, fmt.Sprintf("create dir for %s: %v", path, err)).Write(w)
			return
		}

		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			types.Err(http.StatusInternalServerError, fmt.Sprintf("write file %s: %v", path, err)).Write(w)
			return
		}
	}

	local, err := cfgpkg.Load(app.ConfigPath)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "read local config"))
		return
	}

	mergeIncoming(local, incoming)

	if err := cfgstore.WriteStaging(app.ConfigPath, username, local); err != nil {
		types.Error(log.Logger, w, cfgstore.StagingError(err, "write staging"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}

func mergeIncoming(local, incoming *cfgpkg.Config) {
	if incoming.Tunnels != nil {
		if local.Tunnels == nil {
			local.Tunnels = make(map[string]*cfgpkg.Tunnel)
		}

		for name, t := range incoming.Tunnels {
			local.Tunnels[name] = t
		}
	}

	if incoming.Routing != nil {
		local.Routing = incoming.Routing
	}

	if incoming.HA != nil {
		mergeHA(local, incoming.HA)
	}

	if incoming.Wireguard != nil {
		if local.Wireguard == nil {
			local.Wireguard = make(map[string]*cfgpkg.Wireguard)
		}

		for name, wg := range incoming.Wireguard {
			local.Wireguard[name] = wg
		}
	}

	if incoming.Nftables != nil {
		local.Nftables = incoming.Nftables
	}

	if incoming.Sysctl != nil {
		local.Sysctl = incoming.Sysctl
	}

	if incoming.Users != nil {
		local.Users = incoming.Users
	}

	if incoming.DNS != nil {
		local.DNS = incoming.DNS
	}

	if incoming.Services != nil {
		local.Services = incoming.Services
	}

	if incoming.Logging != nil {
		local.Logging = incoming.Logging
	}

	if incoming.SSH != nil {
		local.SSH = incoming.SSH
	}
}

func mergeHA(local *cfgpkg.Config, incoming *cfgpkg.HA) {
	if local.HA == nil {
		local.HA = &cfgpkg.HA{}
	}

	if incoming.Conntrackd != nil {
		local.HA.Conntrackd = incoming.Conntrackd
	}

	if len(incoming.VRRP) == 0 {
		return
	}

	replaced := map[string]bool{}
	for _, v := range incoming.VRRP {
		replaced[v.Interface] = true
	}

	var kept []*cfgpkg.VRRPInstance
	for _, v := range local.HA.VRRP {
		if !replaced[v.Interface] {
			kept = append(kept, v)
		}
	}

	local.HA.VRRP = append(kept, incoming.VRRP...)
}
