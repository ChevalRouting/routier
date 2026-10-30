package config

import (
	"fmt"
	"strings"
	"testing"
)

func flowErrors(flows *FlowsConfig) []string {
	cfg := &Config{
		Interfaces: map[string]*Interface{"wan": {Device: "eth0"}},
		Monitoring: &MonitoringConfig{Flows: flows},
	}

	var out []string
	validateFlows(cfg, func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) })
	return out
}

func TestValidateFlowsAcceptsDefaults(t *testing.T) {
	if errs := flowErrors(&FlowsConfig{Enabled: true, Collectors: []FlowCollector{{Address: "192.0.2.10"}}}); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateFlowsRejectsBadSettings(t *testing.T) {
	errs := flowErrors(&FlowsConfig{
		Enabled:    true,
		Sampling:   MaxFlowSampling + 1,
		Interfaces: []string{"missing"},
		Source:     "not-an-ip",
		Collectors: []FlowCollector{{Address: "collector.example"}, {Address: "192.0.2.10", Port: 70000}, {Address: "192.0.2.11"}, {Address: "192.0.2.11", Port: DefaultFlowPort}},
		Retention:  FlowRetention{Hours: -1, MaxFlows: -1},
	})

	joined := strings.Join(errs, "\n")
	for _, want := range []string{"sampling", `"missing" is not a configured interface`, "source", `"collector.example" is not a valid IP`, "port must be between", "duplicates collector", "retention.hours", "retention.max_flows"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing error %q in:\n%s", want, joined)
		}
	}
}

func TestValidateFlowsIgnoresDisabled(t *testing.T) {
	if errs := flowErrors(&FlowsConfig{Sampling: -5}); len(errs) != 0 {
		t.Fatalf("disabled flows were validated: %v", errs)
	}
}
