package config

import (
	"net"
	"strconv"
)

const DefaultExporterPort = 9469

var defaultExporterListen = []string{"127.0.0.1", "::1"}

func (m *MonitoringConfig) ExporterEnabled() bool {
	return m != nil && m.Exporter != nil && m.Exporter.Enabled
}

func (e *ExporterConfig) ListenPort() int {
	if e.Port > 0 {
		return e.Port
	}

	return DefaultExporterPort
}

func (e *ExporterConfig) Addresses() []string {
	listen := e.Listen
	if len(listen) == 0 {
		listen = defaultExporterListen
	}

	port := strconv.Itoa(e.ListenPort())
	out := make([]string, len(listen))
	for i, host := range listen {
		out[i] = net.JoinHostPort(host, port)
	}

	return out
}
