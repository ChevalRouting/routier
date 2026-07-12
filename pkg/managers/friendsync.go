package managers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/friends"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/rs/zerolog/log"
)

var defaultHASections = []string{"vrrp", "conntrackd"}

func friendClient(f *config.Friend) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: f.TLSSkipVerify}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

func haMembers(cfg *config.Config) []*config.Friend {
	var members []*config.Friend
	for _, f := range cfg.Friends {
		if f.IsEnabled() && f.HA != nil && f.HA.Enabled && f.HA.Link != nil {
			members = append(members, f)
		}
	}

	return members
}

func SyncSections(f *config.Friend) []string {
	if f.Sync != nil && len(f.Sync.Sections) > 0 {
		return f.Sync.Sections
	}

	return defaultHASections
}

func vrrpInterfaces(cfg *config.Config) map[string]*config.Interface {
	out := map[string]*config.Interface{}
	for name, iface := range cfg.Interfaces {
		if len(iface.VRRP) == 0 {
			continue
		}

		vrrp := make([]*config.VRRPInstance, len(iface.VRRP))
		for i, v := range iface.VRRP {
			cp := *v
			vrrp[i] = &cp
		}

		out[name] = &config.Interface{Select: iface.Select, VRRP: vrrp}
	}

	return out
}

func clusterPeerIPs(cfg *config.Config, target *config.Friend) []string {
	var peers []string
	if cfg.Conntrackd != nil && cfg.Conntrackd.Address != "" {
		peers = append(peers, cfg.Conntrackd.Address)
	}

	for _, f := range haMembers(cfg) {
		if f == target {
			continue
		}

		peers = append(peers, f.HA.Link.Address)
	}

	return peers
}

func localPeerIPs(cfg *config.Config) []string {
	var peers []string
	for _, f := range haMembers(cfg) {
		peers = append(peers, f.HA.Link.Address)
	}

	return peers
}

func conntrackdFor(cfg *config.Config, target *config.Friend) *config.Conntrackd {
	port := target.HA.Link.Port
	if port == 0 && cfg.Conntrackd != nil {
		port = cfg.Conntrackd.Port
	}

	allow := false
	if cfg.Conntrackd != nil {
		allow = cfg.Conntrackd.AllowInbound
	}

	return &config.Conntrackd{
		Interface:    target.HA.Link.Interface,
		Address:      target.HA.Link.Address,
		PeerIPs:      clusterPeerIPs(cfg, target),
		Port:         port,
		AllowInbound: allow,
	}
}

func applyOverrides(out *config.Config, overrides map[string]string) {
	for k, v := range overrides {
		switch k {
		case "conntrackd.interface":
			if out.Conntrackd != nil {
				out.Conntrackd.Interface = v
			}
		case "conntrackd.address":
			if out.Conntrackd != nil {
				out.Conntrackd.Address = v
			}
		case "conntrackd.port":
			if n, err := strconv.Atoi(v); err == nil && out.Conntrackd != nil {
				out.Conntrackd.Port = n
			}
		case "vrrp.priority":
			if n, err := strconv.Atoi(v); err == nil {
				for _, iface := range out.Interfaces {
					for _, vi := range iface.VRRP {
						vi.Priority = n
					}
				}
			}
		}
	}
}

func BuildFriendSyncPayload(cfg *config.Config, f *config.Friend) PushPayload {
	out := config.Config{Version: cfg.Version}
	for _, s := range SyncSections(f) {
		switch s {
		case "vrrp":
			out.Interfaces = vrrpInterfaces(cfg)
		case "conntrackd":
			out.Conntrackd = conntrackdFor(cfg, f)
		}
	}

	if f.Sync != nil {
		applyOverrides(&out, f.Sync.Overrides)
	}

	return PushPayload{Config: &out}
}

func pushToFriend(f *config.Friend, payload PushPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	return pushImportApply(friendClient(f), f, data)
}

func PushWireguardCounterpart(f *config.Friend, version, name string, wg *config.Wireguard) error {
	payload := PushPayload{Config: &config.Config{
		Version:   version,
		Wireguard: map[string]*config.Wireguard{name: wg},
	}}

	return pushToFriend(f, payload)
}

func SyncHA(ctx context.Context, cfg *config.Config, only string) ([]PushResult, error) {
	members := haMembers(cfg)
	if only != "" {
		var target *config.Friend
		for _, f := range members {
			if f.Name == only {
				target = f
			}
		}

		if target == nil {
			return nil, fmt.Errorf("friend %q is not an HA-enabled member", only)
		}

		members = []*config.Friend{target}
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("no HA-enabled friends configured")
	}

	results := make([]PushResult, 0, len(members))
	for _, f := range members {
		res := PushResult{Name: f.Name, URL: f.URL}
		if err := pushToFriend(f, BuildFriendSyncPayload(cfg, f)); err != nil {
			res.Error = err.Error()
			log.Error().Err(err).Str("friend", f.Name).Msg("friend sync: push failed")
		} else {
			log.Info().Str("friend", f.Name).Msg("friend sync: config pushed and applied")
		}

		results = append(results, res)
	}

	if cfg.Conntrackd != nil {
		cfg.Conntrackd.PeerIPs = localPeerIPs(cfg)
	}

	outputs, err := render.All(cfg, render.WithFriends(friends.CachedVars()))
	if err != nil {
		return results, fmt.Errorf("local render: %w", err)
	}

	if _, err := ApplyConfig(ctx, cfg, outputs, ApplyOptions{Source: "ha-sync"}); err != nil {
		return results, fmt.Errorf("local apply: %w", err)
	}

	return results, nil
}
