//go:build linux

package systemtest

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/net/netlink"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestRunNetlinkLogsResult(t *testing.T) {
	old := log.Logger
	defer func() { log.Logger = old }()
	var output bytes.Buffer
	log.Logger = zerolog.New(&output)
	want := errors.New("permission denied")
	calls := 0
	err := netlink.RunCommand("LinkSetMaster", "eth2 master=core", func() error {
		calls++
		return want
	})
	if !errors.Is(err, want) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}

	for _, text := range []string{"netlink execute", "netlink result", "LinkSetMaster", "eth2 master=core", "permission denied", `"success":false`} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("missing %q in %s", text, output.String())
		}
	}

	output.Reset()
	if err := netlink.RunCommand("LinkSetUp", "eth2", successfulNetlinkCommand); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), `"success":true`) {
		t.Fatal(output.String())
	}
}

func successfulNetlinkCommand() error {
	return nil
}
