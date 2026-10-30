package svc

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	VRRPMaster = "MASTER"
	VRRPBackup = "BACKUP"
)

type VRRPTransition struct {
	Instance *config.VRRPInstance
	State    string
}

func (t VRRPTransition) Master() bool {
	return t.State == VRRPMaster
}

type VRRPHook struct {
	Name string
	Run  func(cfg *config.Config, t VRRPTransition) error
}

var vrrpHooks = []VRRPHook{
	{Name: "bgp prepend", Run: bgpPrependHook},
}

var localAddrs = net.InterfaceAddrs

func LiveVRRPStates(cfg *config.Config) (map[string]string, error) {
	out := map[string]string{}
	if cfg == nil || cfg.HA == nil {
		return out, nil
	}

	addrs, err := localAddrs()
	if err != nil {
		return nil, fmt.Errorf("list local addresses: %w", err)
	}

	held := map[string]bool{}
	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok {
			held[ipNet.IP.String()] = true
		}
	}

	for _, inst := range cfg.HA.VRRP {
		out[inst.KeepalivedName()] = VRRPBackup
		if holdsVIP(inst, held) {
			out[inst.KeepalivedName()] = VRRPMaster
		}
	}

	return out, nil
}

func holdsVIP(inst *config.VRRPInstance, held map[string]bool) bool {
	for _, vip := range inst.VIPs {
		if ip := vipIP(vip); ip != nil && held[ip.String()] {
			return true
		}
	}

	return false
}

func vipIP(vip string) net.IP {
	fields := strings.Fields(vip)
	if len(fields) == 0 {
		return nil
	}

	if ip, _, err := net.ParseCIDR(fields[0]); err == nil {
		return ip
	}

	return net.ParseIP(fields[0])
}

func RunVRRPHooks(cfg *config.Config, states map[string]string) error {
	if cfg == nil || cfg.HA == nil {
		return nil
	}

	var errs []error
	for _, inst := range cfg.HA.VRRP {
		state, ok := states[inst.KeepalivedName()]
		if !ok {
			continue
		}

		t := VRRPTransition{Instance: inst, State: state}
		for _, hook := range vrrpHooks {
			if err := hook.Run(cfg, t); err != nil {
				errs = append(errs, fmt.Errorf("vrrp %s %s: %s: %w", inst.KeepalivedName(), state, hook.Name, err))
			}
		}
	}

	return errors.Join(errs...)
}

func SyncVRRPHooks(cfg *config.Config) error {
	if cfg == nil || cfg.HA == nil || len(cfg.HA.VRRP) == 0 {
		return nil
	}

	states, err := LiveVRRPStates(cfg)
	if err != nil {
		return err
	}

	return RunVRRPHooks(cfg, states)
}

func syncVRRPHooks(cfg *config.Config) {
	if err := SyncVRRPHooks(cfg); err != nil {
		log.Warn().Err(err).Msg("vrrp: could not sync transition hooks with the live state")
	}
}
