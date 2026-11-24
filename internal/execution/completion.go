package execution

import "time"

func (a *attempt) recordCleanupTime() {
	a.mu.Lock()
	a.cleanupAt = time.Now()
	a.mu.Unlock()
}

// finalize completes the attempt after an awaited t.Run or from its parent's
// cleanup. Native cleanup alone is too early: testing may still mark cleanup
// panics or data races as failures.
func (a *attempt) finalize() {
	// A sibling cleanup panic can run parent cleanup while this callback is
	// still active. That is process teardown, not native completion of this T.
	if a.t != nil && !a.callbackReturned.Load() {
		return
	}

	a.mu.Lock()
	if a.finalResult != nil {
		a.mu.Unlock()
		return
	}

	// Callback return can precede parallel children and native cleanup. A
	// sibling panic must not finalize this attempt while either is pending.
	if a.t != nil && a.cleanupAt.IsZero() {
		a.mu.Unlock()
		return
	}

	result := a.observeResult()

	a.finalResult = &result
	snapshot := result.clone()
	complete := a.spec.Complete
	a.mu.Unlock()

	if complete != nil {
		complete(snapshot)
	}
}

// result returns an independent snapshot. Not-ready is distinct from passed;
// callers must never wait for a parallel child while keeping its parent alive.
func (a *attempt) result() (Result, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.finalResult == nil {
		return Result{}, false
	}

	return a.finalResult.clone(), true
}

// observeResult reads native status only after completion, while mu is held.
func (a *attempt) observeResult() Result {
	switch {
	case a.t == nil:
		return Result{Kind: ResultNotSelected}

	case a.spec.Decision.Kind == DecisionExclude:
		return Result{Kind: ResultExcluded}

	default:
		status := AttemptPassed
		if a.t.Failed() {
			status = AttemptFailed
		} else if a.t.Skipped() {
			status = AttemptSkipped
		}

		return Result{
			Kind: ResultCompleted,
			Outcome: &Outcome{
				Status:    status,
				TestName:  a.t.Name(),
				StartedAt: a.startedAt,
				CleanupAt: a.cleanupAt,
			},
		}
	}
}

// shouldRetry never waits. A suspended callback is not a passing test, and a
// policy skip must not retry even when its reporting cleanup fails.
func (a *attempt) shouldRetry() bool {
	if a.spec.Decision.Kind != DecisionRun {
		return false
	}

	result, ready := a.result()
	if !ready || result.Kind != ResultCompleted {
		return false
	}

	return result.Outcome.Status == AttemptFailed
}
