package lifecycletest

import (
	"context"
	"github.com/ChevalRouting/routier/tests/testkit"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/svc"
)

func TestApplyDryRunPipeline(t *testing.T) {

	restore := svc.SetCommandRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("status: started\n"), nil
	})
	defer restore()

	cfg, err := config.Load(testkit.WriteConfig(t, t.TempDir(), testkit.FullConfig))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	outputs, err := render.All(cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if _, _, err := managers.ApplyConfig(context.Background(), cfg, outputs, managers.ApplyOptions{DryRun: true}); err != nil {
		t.Fatalf("dry-run apply: %v", err)
	}
}
