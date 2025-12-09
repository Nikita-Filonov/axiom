package teststats

import (
	"sync"
)

type Stats struct {
	mu sync.Mutex

	Total   int
	Passed  int
	Failed  int
	Skipped int
	Flaky   int

	Cases []*CaseResult
}

func NewStats() *Stats {
	return &Stats{}
}

func (s *Stats) Record(cr *CaseResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Total++

	switch cr.Status {
	case StatusPassed:
		s.Passed++
	case StatusFailed:
		s.Failed++
	case StatusSkipped:
		s.Skipped++
	case StatusFlaky:
		s.Flaky++
	}

	s.Cases = append(s.Cases, cr)
}
