package testexplain

import "sync"

type Explainer struct {
	mu           sync.Mutex
	explanations []Explanation
}

func NewExplainer() *Explainer {
	return &Explainer{}
}

func (e *Explainer) Record(explanation Explanation) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.explanations = append(e.explanations, explanation)
}

func (e *Explainer) Snapshot() []Explanation {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]Explanation{}, e.explanations...)
}
