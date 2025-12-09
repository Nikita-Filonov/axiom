package teststats

import (
	"time"

	"github.com/Nikita-Filonov/axiom"
)

const (
	StatusPassed  = "passed"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
	StatusFlaky   = "flaky"
)

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

func NewCaseResult(cfg *axiom.Config) *CaseResult {
	return &CaseResult{
		ID:    cfg.ID,
		Name:  cfg.Name,
		Meta:  cfg.Meta,
		Start: time.Now(),
	}
}

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
