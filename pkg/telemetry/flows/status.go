package flows

import (
	"sync"

	"github.com/ChevalRouting/routier/pkg/types"
)

type statusStore struct {
	mu         sync.Mutex
	running    bool
	settings   Settings
	collectors []*collector
	samples    uint64
	overruns   uint64
	err        string
}

var status statusStore

func (s *statusStore) start(settings Settings, collectors []*collector, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running, s.settings, s.collectors, s.err = true, settings, collectors, ""
	if err != nil {
		s.err = err.Error()
	}
}

func (s *statusStore) stop(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running, s.collectors = false, nil
	if err != nil {
		s.err = err.Error()
	}
}

func (s *statusStore) count(samples, overruns uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.samples += samples
	s.overruns += overruns
}

func collectorStatus(c *collector) types.FlowCollectorStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return types.FlowCollectorStatus{Address: c.addr.String(), Source: c.agent.String(), Sent: c.sent, Errors: c.errors, Error: c.lastErr}
}

func Status(enabled bool) types.FlowStatus {
	status.mu.Lock()
	defer status.mu.Unlock()

	out := types.FlowStatus{
		Enabled:    enabled,
		Running:    status.running,
		Sampling:   status.settings.Sampling,
		Interfaces: append([]string{}, status.settings.Devices...),
		Collectors: []types.FlowCollectorStatus{},
		Samples:    status.samples,
		Overruns:   status.overruns,
		Error:      status.err,
	}

	for _, c := range status.collectors {
		out.Collectors = append(out.Collectors, collectorStatus(c))
	}

	return out
}
