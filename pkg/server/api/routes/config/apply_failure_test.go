package config

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ChevalRouting/routier/pkg/state/failures"
	"github.com/ChevalRouting/routier/pkg/types"
)

func TestApplyFailureResultPreservesValidationDetails(t *testing.T) {
	ve := failures.NewValidationError([]types.ArtifactError{{Dest: "/etc/frr/frr.conf", Line: 5, Message: "bad command"}})
	ve.BundleID = "run-123"
	result := applyFailureResult(&failures.ApplyError{Cause: ve, LogID: "run-123", BundleID: "run-123"})
	if result.Status != "validation_failed" || result.LogID != "run-123" || result.BundleID != "run-123" || result.Errors[0].Line != 5 {
		t.Fatalf("incorrect response: %+v", result)
	}
}

func TestApplyFailureResultPreservesRuntimeAndRollbackCause(t *testing.T) {
	cause := &failures.ApplyError{Cause: errors.New("reload kea-dhcp4: exit 1"), LogID: "run-456"}
	result := applyFailureResult(fmt.Errorf("apply failed: %w; rollback failed: permission denied", cause))
	if result.Status != "apply_failed" || result.LogID != "run-456" || result.BundleID != "" || result.Errors[0].Message != "apply failed: reload kea-dhcp4: exit 1; rollback failed: permission denied" {
		t.Fatalf("incorrect response: %+v", result)
	}
}
