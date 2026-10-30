package svc

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

type orderingRunner struct {
	steps []string
	fail  string
}

func (r *orderingRunner) run(_ context.Context, bin string, args ...string) ([]byte, error) {
	if bin == "rc-service" && args[len(args)-1] == "status" {
		return []byte("status: started"), nil
	}

	step := bin
	if bin == "wg-quick" {
		step += " " + args[0]
	}

	if bin == "nft" {
		step += " " + args[0]
	}

	if bin == "rc-service" && args[0] == "radvd" {
		step += " " + args[len(args)-1]
	}

	r.steps = append(r.steps, step)
	if step == r.fail {
		return []byte("failed"), errors.New("exit status 1")
	}

	return nil, nil
}

func (r *orderingRunner) links(_ *config.Config, _ bool, applyVPN func() error) error {
	r.steps = append(r.steps, "links")
	return applyVPN()
}

func (r *orderingRunner) state(_ *config.Config, _ bool) error {
	r.steps = append(r.steps, "addresses and routes")
	return nil
}

func (r *orderingRunner) tunnels(_ *config.Config, dryRun bool) error {
	if !dryRun {
		r.steps = append(r.steps, "vpn tunnels")
	}

	if r.fail == "vpn tunnels" {
		return errors.New("tunnel failed")
	}

	return nil
}

func setOrderingNetwork(t *testing.T, runner *orderingRunner) {
	t.Helper()
	oldLinks, oldState, oldTunnels := reconcileNetworkLinks, reconcileNetworkState, reconcileVPNTunnels
	reconcileNetworkLinks, reconcileNetworkState, reconcileVPNTunnels = runner.links, runner.state, runner.tunnels
	t.Cleanup(func() {
		reconcileNetworkLinks, reconcileNetworkState, reconcileVPNTunnels = oldLinks, oldState, oldTunnels
	})
}

func orderingConfig() *config.Config {
	return &config.Config{
		BootModules: []string{"wireguard"},
		VPN: config.VPNConfig{Wireguard: map[string]*config.Wireguard{
			"wg-ordering-test": {},
		}},
	}
}

func TestNetworkDependenciesBeforeFirewallAndServices(t *testing.T) {
	runner := &orderingRunner{}
	setOrderingNetwork(t, runner)
	restore := SetCommandRunner(runner.run)
	defer restore()

	names := []string{"modules-load.d/routier.conf", "wireguard/wg-ordering-test.conf", "sysctl/routier.conf", "nftables/routier.nft", "radvd/radvd.conf"}
	if err := ReloadFromOutputs(names, orderingConfig(), false); err != nil {
		t.Fatal(err)
	}

	want := []string{"modprobe", "links", "vpn tunnels", "wg-quick down", "wg-quick up", "sysctl", "addresses and routes", "nft -c", "nft -f", "radvd", "rc-service reload"}
	if !reflect.DeepEqual(runner.steps, want) {
		t.Fatalf("apply order: got %v, want %v", runner.steps, want)
	}
}

func TestWireguardFailureStopsDependentStages(t *testing.T) {
	runner := &orderingRunner{fail: "wg-quick up"}
	setOrderingNetwork(t, runner)
	restore := SetCommandRunner(runner.run)
	defer restore()

	names := []string{"wireguard/wg-ordering-test.conf", "nftables/routier.nft", "radvd/radvd.conf"}
	if err := ReloadFromOutputs(names, orderingConfig(), false); err == nil {
		t.Fatal("want WireGuard failure")
	}

	want := []string{"links", "vpn tunnels", "wg-quick down", "wg-quick up"}
	if !reflect.DeepEqual(runner.steps, want) {
		t.Fatalf("must stop before routes, firewall, and services: %v", runner.steps)
	}
}

func TestNetworkOrderingDryRunIsInert(t *testing.T) {
	runner := &orderingRunner{}
	setOrderingNetwork(t, runner)
	restore := SetCommandRunner(runner.run)
	defer restore()

	names := []string{"modules-load.d/routier.conf", "wireguard/wg-ordering-test.conf", "sysctl/routier.conf", "nftables/routier.nft"}
	if err := ReloadFromOutputs(names, orderingConfig(), true); err != nil {
		t.Fatal(err)
	}

	if len(runner.steps) != 0 {
		t.Fatalf("dry-run must not change network or execute commands: %v", runner.steps)
	}
}

func TestTunnelFailureStopsWireguardAndDependentStages(t *testing.T) {
	runner := &orderingRunner{fail: "vpn tunnels"}
	setOrderingNetwork(t, runner)
	restore := SetCommandRunner(runner.run)
	defer restore()

	names := []string{"wireguard/wg-ordering-test.conf", "nftables/routier.nft"}
	if err := ReloadFromOutputs(names, orderingConfig(), false); err == nil {
		t.Fatal("want tunnel failure")
	}

	want := []string{"links", "vpn tunnels"}
	if !reflect.DeepEqual(runner.steps, want) {
		t.Fatalf("must stop before WireGuard, routes, and firewall: %v", runner.steps)
	}
}
