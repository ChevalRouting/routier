//go:build linux

package netlink

import (
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/dhcpcd"
)

func Reconcile(cfg *config.Config, dryRun bool) error {
	if err := reconcileLinks(cfg, dryRun); err != nil {
		return err
	}

	if err := reconcileAddrs(cfg, dryRun); err != nil {
		return err
	}

	if err := dhcpcd.Reconcile(cfg, dryRun); err != nil {
		return err
	}

	return reconcileRoutes(cfg, dryRun)
}
