package svc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

var commandRunner CommandRunner = execRunner

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	f, err := os.CreateTemp("", "routier-svc-*.log")
	if err != nil {
		return nil, err
	}

	defer os.Remove(f.Name())
	defer f.Close()

	cmd.Stdout = f
	cmd.Stderr = f

	runErr := cmd.Run()

	out, _ := os.ReadFile(f.Name())
	return out, runErr
}

func SetCommandRunner(r CommandRunner) func() {
	prev := commandRunner
	commandRunner = r
	return func() { commandRunner = prev }
}

func runCombined(args []string, timeout time.Duration) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	return commandRunner(ctx, args[0], args[1:]...)
}
