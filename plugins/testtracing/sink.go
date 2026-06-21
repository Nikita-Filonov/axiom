package testtracing

import (
	"sync"

	"github.com/Nikita-Filonov/axiom"
)

type activeSink struct {
	mu     sync.Mutex
	trace  *Trace
	cfg    *axiom.Config
	index  int
	active bool
}

func newActiveSink(trace *Trace, cfg *axiom.Config) *activeSink {
	return &activeSink{trace: trace, cfg: cfg, index: -1, active: true}
}

func (s *activeSink) Append(event axiom.Event) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}

	if s.index < 0 {
		s.index = s.trace.startRecord(s.cfg)
	}
	index := s.index
	s.mu.Unlock()

	s.trace.AppendToRecord(index, event)
}

func (s *activeSink) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.active = false
}
