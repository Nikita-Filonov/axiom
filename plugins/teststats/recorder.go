package teststats

import (
	"sync"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// recorder turns the events of one Config into at most one Attempt.
type recorder struct {
	cfg   *axiom.Config
	stats *Stats

	mu      sync.Mutex
	attempt *Attempt
}

func (r *recorder) observe(event axiom.Event) {
	switch event.Type {
	case axiom.EventTypeCaseStart:
		r.begin("")
	case axiom.EventTypeCaseSkip:
		r.begin(event.Message)
	case axiom.EventTypeCasePanic,
		axiom.EventTypeStepPanic,
		axiom.EventTypeSetupPanic,
		axiom.EventTypeTeardownPanic,
		axiom.EventTypeFixtureSetupFailed,
		axiom.EventTypeFixtureCleanupPanic:
		r.fail(event.Message)
	}
}

func (r *recorder) begin(skipReason string) {
	t := r.cfg.T()

	r.mu.Lock()
	defer r.mu.Unlock()
	if t == nil || r.attempt != nil {
		return
	}

	r.attempt = newAttempt(r.cfg, t, skipReason)
	// Axiom registers no cleanup on the attempt's T before case.start or
	// case.skip, so this one runs after everything the attempt registers.
	t.Cleanup(func() { r.finish(t) })
}

func (r *recorder) fail(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.attempt != nil && r.attempt.Error == "" {
		r.attempt.Error = message
	}
}

func (r *recorder) finish(t *testing.T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.attempt.End = time.Now()
	r.attempt.Duration = r.attempt.End.Sub(r.attempt.Start)
	r.attempt.Status = status(t, r.attempt.Error)
	stored := r.stats.add(*r.attempt)

	if parent := r.cfg.RootT; parent != nil && parent != t {
		// Go marks a panicking t.Cleanup callback or a detected data race as
		// a failure only after the attempt's cleanups have run.
		parent.Cleanup(func() {
			if t.Failed() {
				r.stats.markFailed(stored)
			}
		})
	}
}

func newAttempt(cfg *axiom.Config, t *testing.T, skipReason string) *Attempt {
	return &Attempt{
		RunID: cfg.Execution.ID,
		// A planning Config, which observes a skip applied before parallel
		// retries start, and a Config constructed outside RunCase have attempt zero.
		Number:     max(cfg.Execution.Attempt, 1),
		CaseID:     cfg.Case.ID,
		Name:       cfg.Case.Name,
		TestName:   t.Name(),
		Meta:       cfg.Meta.Copy(),
		SkipReason: skipReason,
		Start:      time.Now(),
	}
}

// status lets a lifecycle failure win because a panic can unwind through the
// attempt's cleanups before Go marks its test failed.
func status(t *testing.T, lifecycleError string) Status {
	switch {
	case t.Failed() || lifecycleError != "":
		return StatusFailed
	case t.Skipped():
		return StatusSkipped
	default:
		return StatusPassed
	}
}
