package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/state/failures"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestApplyFailurePointsToExactLogAndArtifacts(t *testing.T) {
	ve := failures.NewValidationError([]types.ArtifactError{{Dest: "/etc/frr/frr.conf", Line: 5, Message: "bad command"}})
	diagnostic := &failures.ApplyError{Cause: ve, LogID: "run-123", BundleID: "run-123"}
	var output bytes.Buffer
	writeApplyFailure(&output, fmt.Errorf("apply: %w", diagnostic))
	for _, want := range []string{"/etc/frr/frr.conf:5: bad command", "routier history show run-123", "routier failures show run-123", failures.Dir("run-123") + "/rendered"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("missing %q in %s", want, output.String())
		}
	}
}

func TestRuntimeFailureWithoutBundleStillPointsToLog(t *testing.T) {
	var output bytes.Buffer
	writeApplyFailure(&output, &failures.ApplyError{Cause: errors.New("reload kea-dhcp4: exit 1"), LogID: "run-456"})
	if !strings.Contains(output.String(), "reload kea-dhcp4: exit 1") || !strings.Contains(output.String(), "routier history show run-456") {
		t.Fatal(output.String())
	}

	if strings.Contains(output.String(), "routier failures show") {
		t.Fatal("advertised missing artifacts")
	}
}
