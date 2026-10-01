package teststats

import (
	"cmp"
	"slices"
	"sync"
)

// Stats collects attempts. It is safe for concurrent use and its zero value is
// ready to use. Queries return copies. While tests run, they reflect only the
// attempts that have finished; read final results after the parent test ends.
type Stats struct {
	mu       sync.Mutex
	attempts []*Attempt
}

// NewStats returns an empty collector.
func NewStats() *Stats { return &Stats{} }

func (s *Stats) add(a Attempt) *Attempt {
	stored := a.clone()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts = append(s.attempts, &stored)

	return &stored
}

func (s *Stats) markFailed(a *Attempt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.Status = StatusFailed
}

// Attempts returns all attempts ordered by start time, then run ID and number.
func (s *Stats) Attempts() []Attempt {
	s.mu.Lock()
	attempts := make([]Attempt, len(s.attempts))
	for i, a := range s.attempts {
		attempts[i] = a.clone()
	}
	s.mu.Unlock()

	slices.SortStableFunc(attempts, func(a, b Attempt) int {
		return cmp.Or(
			a.Start.Compare(b.Start),
			cmp.Compare(a.RunID, b.RunID),
			cmp.Compare(a.Number, b.Number),
		)
	})

	return attempts
}

// Filter returns a collector with the attempts that match. match runs on
// copies without holding the lock, so it may query s. Filtering by status
// drops the attempts that make a run flaky; filter by case identity or
// metadata to keep whole runs.
func (s *Stats) Filter(match func(Attempt) bool) *Stats {
	out := NewStats()
	for _, a := range s.Attempts() {
		if match(a.clone()) {
			out.add(a)
		}
	}

	return out
}
