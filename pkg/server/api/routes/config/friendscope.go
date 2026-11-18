package config

import (
	"errors"
	"fmt"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
)

var errFriendScope = errors.New("friend is not allowed to manage this node")

func friendTag(sender *cfgpkg.Friend, tag string) bool {
	return tag != "" && (tag == sender.Name || tag == sender.Hostname)
}

func scopeFriendPayload(local *cfgpkg.Config, sender *cfgpkg.Friend, payload *managers.PushPayload) error {
	if sender.Manage {
		return nil
	}

	if len(payload.Files) > 0 {
		return fmt.Errorf("%w: template files", errFriendScope)
	}

	in := payload.Config
	scoped := &cfgpkg.Config{Version: in.Version}

	for name, iface := range in.Interfaces {
		if iface == nil || !friendTag(sender, iface.Friend) {
			return fmt.Errorf("%w: interface %s is not tagged for this friend", errFriendScope, name)
		}

		if existing := local.Interfaces[name]; existing != nil && !friendTag(sender, existing.Friend) {
			return fmt.Errorf("%w: interface %s already exists", errFriendScope, name)
		}
	}

	scoped.Interfaces = in.Interfaces

	for name, wg := range in.VPN.Wireguard {
		if wg == nil || !friendTag(sender, wg.Friend) {
			return fmt.Errorf("%w: wireguard %s is not tagged for this friend", errFriendScope, name)
		}

		if existing := local.VPN.Wireguard[name]; existing != nil && !friendTag(sender, existing.Friend) {
			return fmt.Errorf("%w: wireguard %s already exists", errFriendScope, name)
		}

		if err := wireguardHooksAbsent(name, wg); err != nil {
			return err
		}
	}

	scoped.VPN.Wireguard = in.VPN.Wireguard

	if in.HA != nil {
		ha, err := scopeFriendHA(local, sender, in.HA)
		if err != nil {
			return err
		}

		scoped.HA = ha
	}

	payload.Config = scoped

	return nil
}

func wireguardHooksAbsent(name string, wg *cfgpkg.Wireguard) error {
	if len(wg.PreUp)+len(wg.PostUp)+len(wg.PreDown)+len(wg.PostDown) > 0 || wg.PrivateKeyFile != "" {
		return fmt.Errorf("%w: wireguard %s carries hooks or key files", errFriendScope, name)
	}

	for _, peer := range wg.Peers {
		if peer.PresharedKeyFile != "" {
			return fmt.Errorf("%w: wireguard %s peer %s reads a key file", errFriendScope, name, peer.Name)
		}
	}

	return nil
}

func scopeFriendHA(local *cfgpkg.Config, sender *cfgpkg.Friend, in *cfgpkg.HA) (*cfgpkg.HA, error) {
	if sender.HA != nil && sender.HA.Enabled {
		return &cfgpkg.HA{VRRP: in.VRRP, Conntrackd: in.Conntrackd}, nil
	}

	if in.Conntrackd != nil {
		return nil, fmt.Errorf("%w: conntrackd requires an HA link with this friend", errFriendScope)
	}

	for _, v := range in.VRRP {
		if v == nil || !friendTag(sender, v.Friend) {
			return nil, fmt.Errorf("%w: vrrp instance is not tagged for this friend", errFriendScope)
		}

		if local.HA == nil {
			continue
		}

		for _, existing := range local.HA.VRRP {
			if existing.Interface == v.Interface && !friendTag(sender, existing.Friend) {
				return nil, fmt.Errorf("%w: vrrp on %s already exists", errFriendScope, v.Interface)
			}
		}
	}

	return &cfgpkg.HA{VRRP: in.VRRP}, nil
}
