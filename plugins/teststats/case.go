package teststats

import (
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// Status values classify a CaseResult.
const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
	StatusFlaky   = "flaky"
)

// CaseResult holds an observed result, timing, and metadata. The Plugin records
// one result when an attempt reaches its AfterTest hook, preserving retry
// outcomes.
type CaseResult struct {
	ID       string
	Name     string
	Attempts int
	Duration time.Duration
	Status   string
	Error    error
	Start    time.Time
	End      time.Time
	Meta     axiom.Meta
}

// NewCaseResult starts a result record from the current case configuration.
func NewCaseResult(cfg *axiom.Config) *CaseResult {
	return &CaseResult{
		ID:    cfg.Case.ID,
		Name:  cfg.Case.Name,
		Meta:  cfg.Meta.Copy(),
		Start: time.Now(),
	}
}

// Finalize fills the status and timing from cfg and the supplied attempt count.
func (r *CaseResult) Finalize(cfg *axiom.Config, attempts int) {
	r.Attempts = attempts
	r.End = time.Now()
	r.Duration = r.End.Sub(r.Start)

	if cfg.Skip.Enabled {
		r.Status = StatusSkipped
		return
	}

	if !cfg.SubT.Failed() {
		if attempts > 1 {
			r.Status = StatusFlaky
		} else {
			r.Status = StatusPassed
		}
		return
	}

	r.Status = StatusFailed
}
