package config

import (
	"encoding/json"
	"io"
	"net/http"

	appctx "github.com/ChevalRouting/routier/pkg/api/app"
	"github.com/ChevalRouting/routier/pkg/api/cfgstore"
	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// GetSection godoc
// @Summary  Get one config section
// @Tags config
// @Produce json
// @Param section path string true "section name"
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/config/{section} [get]
func GetSection(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	section := chi.URLParam(r, "section")
	cfg, err := cfgstore.Read(app.ConfigPath, appctx.UsernameFromContext(r.Context()))
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	var data any
	switch section {
	case "interfaces":
		data = cfg.Interfaces
	case "tunnels":
		data = cfg.Tunnels
	case "routing":
		data = cfg.Routing
	case "wireguard":
		data = cfg.Wireguard
	case "nftables":
		data = cfg.Nftables
	case "sysctl":
		data = cfg.Sysctl
	case "dns":
		data = cfg.DNS
	case "users":
		data = cfg.Users
	case "services":
		data = cfg.Services
	case "hostname":
		data = cfg.Hostname
	case "logging":
		data = cfg.Logging
	case "conntrackd":
		data = cfg.Conntrackd
	case "ssh":
		data = cfg.SSH
	case "vrfs":
		data = cfg.VRFs
	case "monitoring":
		data = cfg.Monitoring
	case "dhcp":
		data = cfg.DHCP
	default:
		types.Err(http.StatusNotFound, "unknown section: "+section).Write(w)
		return
	}

	types.OK(w, data)
}

// PutSection godoc
// @Summary  Replace one config section
// @Tags config
// @Accept json
// @Produce json
// @Param section path string true "section name"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/config/{section} [put]
func PutSection(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	section := chi.URLParam(r, "section")
	username := appctx.UsernameFromContext(r.Context())

	cfg, err := cfgstore.Read(app.ConfigPath, username)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read config"))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		types.Err(http.StatusBadRequest, "failed to read request body").Write(w)
		return
	}

	if appErr := applySectionBody(cfg, section, body); appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	if appErr := cfgstore.ValidationError(cfg); appErr != nil {
		types.Error(log.Logger, w, appErr)
		return
	}

	if err := cfgstore.WriteStaging(app.ConfigPath, username, cfg); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to write staging config"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}

func applySectionBody(cfg *cfgpkg.Config, section string, body []byte) *types.AppError {
	unmarshalErr := func(err error) *types.AppError {
		return types.Errorf(http.StatusBadRequest, "invalid %s data: %v", section, err)
	}

	switch section {
	case "interfaces":
		var v map[string]*cfgpkg.Interface
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Interfaces = v
	case "tunnels":
		var v map[string]*cfgpkg.Tunnel
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Tunnels = v
	case "routing":
		var v *cfgpkg.Routing
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Routing = v
	case "wireguard":
		var v map[string]*cfgpkg.Wireguard
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Wireguard = v
	case "nftables":
		var v cfgpkg.NftablesConfig
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Nftables = &v
	case "sysctl":
		var v map[string]string
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Sysctl = v
	case "dns":
		var v *cfgpkg.DNS
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.DNS = v
	case "users":
		var v map[string]*cfgpkg.User
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Users = v
	case "services":
		var v map[string]*cfgpkg.Service
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Services = v
	case "hostname":
		var v string
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Hostname = v
	case "logging":
		var v *cfgpkg.Logging
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Logging = v
	case "conntrackd":
		var v *cfgpkg.Conntrackd
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Conntrackd = v
	case "ssh":
		var v *cfgpkg.SSH
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.SSH = v
	case "vrfs":
		var v map[string]*cfgpkg.VRFConfig
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.VRFs = v
	case "monitoring":
		var v *cfgpkg.MonitoringConfig
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.Monitoring = v
	case "dhcp":
		var v *cfgpkg.DHCP
		if err := json.Unmarshal(body, &v); err != nil {
			return unmarshalErr(err)
		}

		cfg.DHCP = v
	default:
		return types.Errorf(http.StatusNotFound, "unknown section: %s", section)
	}

	return nil
}
