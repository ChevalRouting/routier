package render

import (
	"strings"
	"testing"

	"github.com/ChevalRouting/routier/pkg/config"
)

func lldpOutputs(t *testing.T, cfg *config.Config) map[string]string {
	t.Helper()
	out, err := renderLLDP(cfg)
	if err != nil {
		t.Fatalf("renderLLDP: %v", err)
	}

	byName := map[string]string{}
	for _, o := range out {
		byName[o.Name] = o.Content
	}

	return byName
}

func TestRenderLLDPDisabled(t *testing.T) {
	cfg := &config.Config{Monitoring: &config.MonitoringConfig{LLDP: &config.LLDPConfig{Enabled: false}}}
	if out, err := renderLLDP(cfg); err != nil || out != nil {
		t.Fatalf("renderLLDP disabled = %v, %v; want nil, nil", out, err)
	}

	if out, err := renderLLDP(&config.Config{}); err != nil || out != nil {
		t.Fatalf("renderLLDP nil monitoring = %v, %v; want nil, nil", out, err)
	}
}

func TestRenderLLDPReceiveOnlyWithCDP(t *testing.T) {
	cfg := &config.Config{Hostname: "edge-router", Monitoring: &config.MonitoringConfig{LLDP: &config.LLDPConfig{
		Enabled:    true,
		CDP:        true,
		Transmit:   false,
		Interfaces: []string{"eth1", "eth2"},
	}}}

	out := lldpOutputs(t, cfg)
	confd, ok := out[lldpdOptsName]
	if !ok {
		t.Fatalf("missing %s output", lldpdOptsName)
	}

	if !strings.Contains(confd, `LLDPD_OPTS="-c -r -I eth1,eth2"`) {
		t.Errorf("conf.d opts wrong:\n%s", confd)
	}

	conf, ok := out[lldpdConfName]
	if !ok {
		t.Fatalf("missing %s output", lldpdConfName)
	}

	if !strings.Contains(conf, `configure system description "Routier"`) {
		t.Errorf("lldpd.conf missing Routier description:\n%s", conf)
	}

	if !strings.Contains(conf, `configure system hostname "edge-router"`) {
		t.Errorf("lldpd.conf missing hostname identity:\n%s", conf)
	}
}

func TestRenderLLDPTransmitNoCDP(t *testing.T) {
	cfg := &config.Config{Monitoring: &config.MonitoringConfig{LLDP: &config.LLDPConfig{
		Enabled:  true,
		Transmit: true,
	}}}

	confd := lldpOutputs(t, cfg)[lldpdOptsName]
	if !strings.Contains(confd, `LLDPD_OPTS=""`) {
		t.Errorf("expected empty opts for transmit without cdp:\n%s", confd)
	}
}
