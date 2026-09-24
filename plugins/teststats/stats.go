package teststats

import (
	"sync"
)

// Stats collects attempt results so callers can inspect retry history or
// aggregate results by case. Record is safe for concurrent use.
type Stats struct {
	mu sync.Mutex

	Total   int
	Passed  int
	Failed  int
	Skipped int
	Flaky   int

	Cases []*CaseResult
}

// NewStats returns an empty case result collector.
func NewStats() *Stats {
	return &Stats{}
}

// Record adds a result and updates counts of recorded statuses.
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
