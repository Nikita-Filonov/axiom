package testjunit

import (
	"sync"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// recorder owns the lifecycle of one Config, including a planning policy skip.
type recorder struct {
	cfg      *axiom.Config
	reporter *Reporter
	config   Config
	mu       sync.Mutex
	attempt  *attempt
	failed   bool
	finished bool
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
		SuiteName: r.config.SuiteName,
		TestName:  t.Name(), SkipReason: skipReason, Start: time.Now(),
	}
	// Register before the body, so its hooks, fixtures, subtests and cleanups
	// finish first. The parent cleanup rechecks failures from later cleanups.
	t.Cleanup(func() { r.finish(t) })
}

func (r *recorder) fail(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.attempt != nil && !r.finished && !r.failed {
		r.failed = true
		r.attempt.Error = message
	}
}

func (r *recorder) finish(t testOutcome) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.attempt == nil || r.finished {
		return
	}
	r.finished = true
	result := *r.attempt
	result.Duration = time.Since(result.Start)
	result.Status = outcome(t, r.failed)
	stored := r.reporter.add(result)
	if parent := r.cfg.RootT; parent != nil && parent != t {
		// Checking another T while it finishes can race with testing's own
		// finalization. Its parent's cleanup runs after the child has finished.
		parent.Cleanup(func() { r.reconcile(t, stored) })
	}
}

func (r *recorder) reconcile(t testOutcome, result *attempt) {
	if t.Failed() {
		r.reporter.markFailed(result)
	}
}
