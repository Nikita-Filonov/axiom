package testexplain

import "sync"

// Explainer stores configuration snapshots for later inspection.
// Its methods are safe for concurrent use.
type Explainer struct {
	mu           sync.Mutex
	explanations []Explanation
}

// NewExplainer returns an empty snapshot collector.
func NewExplainer() *Explainer {
	return &Explainer{}
}

// Record appends a snapshot to the collector.
func (e *Explainer) Record(explanation Explanation) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.explanations = append(e.explanations, copyExplanation(explanation))
}

// Snapshot returns a copy of the collected snapshot slice.
func (e *Explainer) Snapshot() []Explanation {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make([]Explanation, len(e.explanations))
	for i, explanation := range e.explanations {
		result[i] = copyExplanation(explanation)
	}
	return result
}
