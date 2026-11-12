package teststats

import (
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// recorder turns the events of one Config into at most one Attempt.
type recorder struct {
	cfg *axiom.Config
	// Values are nil until finish stores an attempt in the owning collector.
	collectors map[*Stats]*Attempt
	outputs    []*outputWriter

	mu            sync.Mutex
	attempt       *Attempt
	artifactRunID string
	revision      uint64
}

func (r *recorder) addCollector(stats *Stats) {
	if _, exists := r.collectors[stats]; !exists {
		r.collectors[stats] = nil
	}
}

func (r *recorder) addOutput(output *outputWriter) {
	if output == nil {
		return
	}
	for _, existing := range r.outputs {
		if existing.dir == output.dir {
			return
		}
	}
	r.outputs = append(r.outputs, output)
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
	if t == nil || r.attempt != nil {
		r.mu.Unlock()
		return
	}

	r.attempt = newAttempt(r.cfg, t, skipReason)
	r.artifactRunID = r.attempt.RunID
	if r.artifactRunID == "" && len(r.outputs) > 0 {
		r.artifactRunID = rand.Text()
	}
	// Axiom registers no cleanup on the attempt's T before case.start or
	// case.skip, so this one runs after everything the attempt registers.
	t.Cleanup(func() { r.finish(t) })
	r.mu.Unlock()
	r.publish(stateRunning)
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

	r.attempt.End = time.Now()
	r.attempt.Duration = r.attempt.End.Sub(r.attempt.Start)
	r.attempt.Status = status(t, r.attempt.Error)
	for stats := range r.collectors {
		r.collectors[stats] = stats.add(*r.attempt)
	}

	if parent := r.cfg.RootT; parent != nil && parent != t {
		// Go marks a panicking t.Cleanup callback or a detected data race as
		// a failure only after the attempt's cleanups have run.
		parent.Cleanup(func() { r.finalize(t) })
	}
	r.mu.Unlock()
	// Without a parent there is no later observation point. In that case the
	// artifact remains finished, rather than claiming an unobserved final status.
	r.publish(stateFinished)
}

func (r *recorder) finalize(t *testing.T) {
	r.mu.Lock()
	if t.Failed() {
		r.attempt.Status = StatusFailed
		for stats, stored := range r.collectors {
			stats.markFailed(stored)
		}
	}
	r.mu.Unlock()
	r.publish(stateFinalized)
}

func (r *recorder) publish(state attemptState) {
	if len(r.outputs) == 0 {
		return
	}
	r.mu.Lock()
	r.revision++
	record := newAttemptArtifact(r.artifactRunID, *r.attempt, state, r.revision)
	r.mu.Unlock()
	// Lifecycle callbacks publish in order. I/O and user error handlers run
	// without the recorder lock, so handlers can query stats or emit events.
	for _, output := range r.outputs {
		output.publish(record)
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
