package failures

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/types"
)

func sampleOutputs() []render.Output {
	return []render.Output{{Name: "frr/frr.conf", Dest: "/etc/frr/frr.conf", Content: "hostname r1\n"}}
}

func TestSaveListGetReadArtifact(t *testing.T) {
	defer SetDir(t.TempDir())()

	cfg := &config.Config{Hostname: "r1", Version: "v3.0.0"}
	errs := []types.ArtifactError{{Tool: "vtysh", Dest: "/etc/frr/frr.conf", Line: 1, Message: "bad"}}
	if err := Save("20260101-000000-abcd", "cli", "snap1", cfg, sampleOutputs(), errs); err != nil {
		t.Fatal(err)
	}

	metas := List()
	if len(metas) != 1 || metas[0].ID != "20260101-000000-abcd" {
		t.Fatalf("unexpected list: %+v", metas)
	}

	m, err := Get("20260101-000000-abcd")
	if err != nil {
		t.Fatal(err)
	}

	if m.SnapID != "snap1" || len(m.Errors) != 1 || len(m.Artifacts) != 1 || m.Artifacts[0] != "/etc/frr/frr.conf" {
		t.Fatalf("unexpected meta: %+v", m)
	}

	data, err := ReadArtifact(m.ID, "/etc/frr/frr.conf")
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != "hostname r1\n" {
		t.Fatalf("unexpected artifact content: %q", data)
	}
}

func TestReadArtifactRejectsTraversal(t *testing.T) {
	defer SetDir(t.TempDir())()

	cfg := &config.Config{Hostname: "r1", Version: "v3.0.0"}
	if err := Save("id1", "cli", "", cfg, sampleOutputs(), nil); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadArtifact("id1", "../../../../etc/passwd"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestValidationErrorMessage(t *testing.T) {
	ve := NewValidationError([]types.ArtifactError{{Dest: "/etc/frr/frr.conf", Line: 5, Message: "bad cmd"}})
	if !strings.Contains(ve.Error(), "/etc/frr/frr.conf:5: bad cmd") {
		t.Fatalf("unexpected message: %q", ve.Error())
	}
}
