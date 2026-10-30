package install

import (
	"errors"
	"strings"
	"testing"
)

func TestJobLifecycle(t *testing.T) {
	j := &job{state: "idle"}
	if !j.begin("/dev/sda") {
		t.Fatal("idle job refused to start")
	}

	if j.begin("/dev/sdb") {
		t.Fatal("running job started twice")
	}

	j.finish(errors.New("exit status 1"))
	if got := j.snapshot(); got.State != "failed" || got.Error != "exit status 1" {
		t.Fatalf("snapshot = %#v", got)
	}

	if !j.begin("/dev/sdb") {
		t.Fatal("failed job could not be retried")
	}

	j.finish(nil)
	if j.begin("/dev/sda") {
		t.Fatal("completed job started again")
	}
}

func TestCollectStripsColorsAndCapsLog(t *testing.T) {
	input := "\x1b[1;34m==> Wiping /dev/sda...\x1b[0m\n\n" + strings.Repeat("line\n", logLines+5)

	j := &job{state: "running"}
	j.collect(strings.NewReader(input), nil, make(chan struct{}))

	got := j.snapshot().Log
	if len(got) != logLines {
		t.Fatalf("kept %d lines, want %d", len(got), logLines)
	}

	short := &job{state: "running"}
	short.collect(strings.NewReader("\x1b[1;34m==> Wiping /dev/sda...\x1b[0m\n"), nil, make(chan struct{}))
	if line := short.snapshot().Log[0]; line != "==> Wiping /dev/sda..." {
		t.Fatalf("line = %q", line)
	}
}
