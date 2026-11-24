package testjunit

import (
	"sync"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// recorder owns the lifecycle of one attempt Config, including a policy skip.
type recorder struct {
	cfg       *axiom.Config
	reporter  *Reporter
	suiteName string

	mu      sync.Mutex
	failed  bool
	stored  *attempt
	attempt *attempt
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
	r.attempt = &attempt{
		SuiteName:  r.suiteName,
		TestName:   t.Name(),
		SkipReason: skipReason,
		Start:      time.Now(),
	}
	// Register before the body, so its hooks, fixtures, subtests and cleanups
	// finish first. The final sink confirms status after native completion.
	t.Cleanup(func() { r.finish(t) })
}

func (r *recorder) fail(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.attempt != nil && r.stored == nil && !r.failed {
		r.failed = true
		r.attempt.Error = message
	}
}

func (r *recorder) finish(t testOutcome) {
	r.mu.Lock()
	if r.attempt == nil || r.stored != nil {
		r.mu.Unlock()
		return
	}
	r.attempt.Duration = time.Since(r.attempt.Start)
	r.attempt.Status = outcome(t, r.failed)
	r.stored = r.reporter.add(*r.attempt)
	r.mu.Unlock()

	// Persist before the parent finishes: a later panic or process timeout must
	// not discard attempts whose cleanup has already completed.
	r.reporter.publish()
}

// reconcile confirms an existing record without reading a finished testing.T.
// Final timing excludes waiting for the parent to deliver a parallel result.
func (r *recorder) reconcile(attempt axiom.Attempt) {
	if attempt.Outcome == nil {
		return
	}

	r.mu.Lock()
	if r.stored == nil {
		r.mu.Unlock()
		return
	}

	confirmed := *r.attempt
	confirmed.Start = attempt.Outcome.StartedAt
	confirmed.Duration = attempt.Outcome.CleanupAt.Sub(confirmed.Start)
	switch {
	case r.failed || attempt.Outcome.Status == axiom.AttemptFailed:
		confirmed.Status = statusFailed
	case attempt.Outcome.Status == axiom.AttemptSkipped:
		confirmed.Status = statusSkipped
	default:
		confirmed.Status = statusPassed
	}
	changed := r.reporter.update(r.stored, confirmed)
	r.mu.Unlock()

	if changed {
		r.reporter.publish()
	}
}
