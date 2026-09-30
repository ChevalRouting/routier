package vtysh

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

var runVtysh = func(cmd string) ([]byte, error) {
	return exec.Command("vtysh", "-c", cmd).Output()
}

func SetRunner(fn func(cmd string) ([]byte, error)) func() {
	prev := runVtysh
	runVtysh = fn
	return func() { runVtysh = prev }
}

func BGPSummaryJSON() ([]byte, error) {
	return runVtysh("show bgp summary json")
}

func OSPFNeighborsJSON() ([]byte, error) {
	return runVtysh("show ip ospf neighbor json")
}

func RoutesJSON(af string) ([]byte, error) {
	return runVtysh("show " + af + " route json")
}

func WaitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := exec.CommandContext(ctx, "vtysh", "-c", "show version").Run()
		cancel()
		if err == nil {
			return nil
		}

		time.Sleep(time.Second)
	}

	return fmt.Errorf("FRR daemons did not become ready within %v", timeout)
}
