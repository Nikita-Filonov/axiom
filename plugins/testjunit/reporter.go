package testjunit

import (
	"cmp"
	"slices"
	"sync"
)

// Reporter collects attempts and exports a snapshot after the tests finish.
// Its collection and export methods are safe to call concurrently, but an
// export made while tests are running contains the results collected so far.
// Its zero value is ready to use.
// A Reporter must not be copied after first use.
type Reporter struct {
	mu       sync.Mutex
	attempts []*attempt
}

// NewReporter creates a JUnit reporter with an empty result set.
func NewReporter() *Reporter {
	return &Reporter{}
}

func (r *Reporter) add(result attempt) *attempt {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts = append(r.attempts, &result)
	return &result
}

func (r *Reporter) markFailed(result *attempt) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result.Status = statusFailed
}

func (r *Reporter) snapshot() reportSnapshot {
	r.mu.Lock()
	results := make(reportSnapshot, len(r.attempts))
	for i, result := range r.attempts {
		results[i] = *result
	}
	r.mu.Unlock()
	slices.SortStableFunc(results, func(a, b attempt) int {
		return cmp.Or(a.Start.Compare(b.Start), cmp.Compare(a.TestName, b.TestName))
	})
	return results
}
