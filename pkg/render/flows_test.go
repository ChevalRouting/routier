package render

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func flowsConfig(flows *config.FlowsConfig) *config.Config {
	return &config.Config{
		Interfaces: map[string]*config.Interface{
			"wan": {Device: "eth0"},
			"lan": {Device: "br-lan"},
		},
		Monitoring: &config.MonitoringConfig{Flows: flows},
	}
}

func TestFlowsTableDisabled(t *testing.T) {
	if got := renderFlowsTable(flowsConfig(&config.FlowsConfig{})); got != "" {
		t.Fatalf("disabled flows rendered a table:\n%s", got)
	}

	if got := renderFlowsTable(&config.Config{}); got != "" {
		t.Fatalf("missing monitoring rendered a table:\n%s", got)
	}
}

func TestFlowsTableSamplesForwardAndInput(t *testing.T) {
	got := renderFlowsTable(flowsConfig(&config.FlowsConfig{Enabled: true, Sampling: 512}))

	rule := `iifname { "br-lan", "eth0" } numgen random mod 512 0 log group 6343 snaplen 128 comment "routier: sflow sampling"`
	for _, want := range []string{
		"table inet routier_flows {",
		"chain sample_forward {",
		"type filter hook forward priority filter - 10; policy accept;",
		"chain sample_input {",
		"type filter hook input priority filter - 10; policy accept;",
		rule,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("flows table is missing %q:\n%s", want, got)
		}
	}
}

func TestFlowsTableSelectedInterfacesWithoutSampling(t *testing.T) {
	got := renderFlowsTable(flowsConfig(&config.FlowsConfig{Enabled: true, Sampling: 1, Interfaces: []string{"wan"}}))
	if !strings.Contains(got, `iifname "eth0" log group 6343`) {
		t.Fatalf("a 1:1 rate on one interface should log every packet from it:\n%s", got)
	}
}
