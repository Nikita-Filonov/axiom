package execution

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// attempt owns lifecycle state. The mutex protects late native completion
// and callbacks moved to worker goroutines by wrappers. User observers are
// always called outside it. A finalized run ignores late wrapper activity.
type attempt struct {
	t    *testing.T
	spec Spec

	callbackReturned atomic.Bool

	mu          sync.Mutex
	startedAt   time.Time
	cleanupAt   time.Time
	finalResult *Result
}

// attemptTeardownPanic distinguishes callback failures: body/before panics
// become test failures, whereas after/release panics propagate.
type attemptTeardownPanic struct{ value any }

// run creates the native subtest and arranges completion observation.
// It may return before a parallel attempt finishes; parent cleanup observes
// that attempt's final result. Call run once per attempt.
func (a *attempt) run(parent *testing.T) {
	parent.Cleanup(a.finalize)

	parent.Run(a.spec.Name, func(t *testing.T) {
		a.t = t
		defer a.callbackReturned.Store(true)

		if a.spec.Bind != nil {
			a.spec.Bind(t)
		}

		t.Cleanup(a.recordCleanupTime)

		if a.spec.Decision.Kind == DecisionExclude {
			t.Skip(a.spec.Decision.Reason)
		}

		if a.spec.Parallel && a.spec.Decision.Kind == DecisionRun {
			t.Parallel()
		}

		a.execute(t)
	})

	// In a correctly scoped t.Run call the parent cannot return and resume a
	// parallel child while this call is still running. A returned callback
	// therefore means Run awaited native completion, including all cleanups.
	if a.t == nil || a.callbackReturned.Load() {
		a.finalize()
	}
}

func (a *attempt) execute(t *testing.T) {
	a.mu.Lock()
	a.startedAt = time.Now()
	a.mu.Unlock()

	a.emit(Event{Type: AttemptStarted})
	defer func() {
		if value := recover(); value != nil {
			if p, ok := value.(attemptTeardownPanic); ok {
				panic(p.value)
			}

			a.emit(Event{
				Type:    AttemptPanicRecovered,
				Message: fmt.Sprint(value),
			})
			t.Helper()
			t.Errorf("panic in test %q: %v", a.spec.Name, value)
		}

		a.emit(Event{Type: AttemptExecutionFinished})
	}()

	var called atomic.Bool
	next := func() {
		if !called.CompareAndSwap(false, true) {
			panic("execution: test continuation called more than once")
		}

		if a.spec.Decision.Kind == DecisionSkip {
			t.Skip(a.spec.Decision.Reason)
		}

		defer a.teardown()

		if a.spec.Before != nil {
			a.spec.Before()
		}

		if a.spec.Body != nil {
			a.spec.Body()
		}
	}

	if a.spec.Around == nil {
		next()
	} else {
		a.spec.Around(next)
	}
}

func (a *attempt) teardown() {
	defer func() {
		if value := recover(); value != nil {
			panic(attemptTeardownPanic{value})
		}
	}()

	if a.spec.Release != nil {
		defer a.spec.Release()
	}

	if a.spec.After != nil {
		a.spec.After()
	}
}

func (a *attempt) emit(event Event) {
	a.mu.Lock()
	if a.finalResult != nil {
		a.mu.Unlock()
		return
	}

	event.Decision = a.spec.Decision
	a.mu.Unlock()

	a.dispatch(event)
}

func (a *attempt) dispatch(event Event) {
	if a.spec.Observe != nil {
		a.spec.Observe(event)
	}
}
