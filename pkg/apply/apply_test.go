package apply

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChevalRouting/routier/pkg/render"
)

func TestModeFor(t *testing.T) {
	cases := map[string]os.FileMode{
		"/etc/nftables.d/routier.nft":          0644,
		"/etc/wireguard/wg0.conf":              0600,
		"/var/lib/routier/last-applied.yml":    0600,
		"/etc/routier/out/interfaces/setup.sh": 0755,
	}
	for dest, want := range cases {
		if got := modeFor(dest); got != want {
			t.Errorf("modeFor(%q) = %o, want %o", dest, got, want)
		}
	}
}

func TestTakeSnapshotIncludesAlso(t *testing.T) {
	dir := t.TempDir()
	outDest := filepath.Join(dir, "rendered.conf")
	alsoDest := filepath.Join(dir, "last-applied.yml")
	missingDest := filepath.Join(dir, "absent.yml")

	if err := os.WriteFile(outDest, []byte("out"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(alsoDest, []byte("prev-config"), 0600); err != nil {
		t.Fatal(err)
	}

	snap := takeSnapshot(
		[]render.Output{{Dest: outDest, Content: "out"}},
		[]string{alsoDest, missingDest},
	)

	if snap.Files[outDest] != "out" {
		t.Errorf("output not captured: %q", snap.Files[outDest])
	}

	if snap.Files[alsoDest] != "prev-config" {
		t.Errorf("extra file not captured: %q", snap.Files[alsoDest])
	}

	if _, ok := snap.Files[missingDest]; !ok || snap.Files[missingDest] != "" {
		t.Errorf("absent extra file should snapshot as empty (delete-on-rollback), got %q ok=%v", snap.Files[missingDest], ok)
	}
}
