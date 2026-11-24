package execution

import "time"

func (a *attempt) recordCleanupTime() {
	a.mu.Lock()
	a.cleanupAt = time.Now()
	a.mu.Unlock()
}

// finalize must be called only after native completion, either after an
// awaited t.Run or from its parent's cleanup. Native cleanup alone is too
// early: testing may still mark cleanup panics or data races as failures.
func (a *attempt) finalize() {
	a.mu.Lock()
	if a.finalResult != nil {
		a.mu.Unlock()
		return
	}

	result := a.observeResult()

	a.finalResult = &result
	snapshot := result.clone()
	event := Event{
		Type:     terminalEventType(result.Kind),
		Result:   &snapshot,
		Decision: a.spec.Decision,
	}
	a.mu.Unlock()

	a.dispatch(event)
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
		return Result{Kind: AttemptResultNotSelected}

	case a.spec.Decision.Kind == DecisionExclude:
		return Result{Kind: AttemptResultExcluded}

	default:
		status := AttemptPassed
		if a.t.Failed() {
			status = AttemptFailed
		} else if a.t.Skipped() {
			status = AttemptSkipped
		}

		return Result{
			Kind: AttemptResultCompleted,
			Outcome: &Outcome{
				Status:    status,
				TestName:  a.t.Name(),
				StartedAt: a.startedAt,
				CleanupAt: a.cleanupAt,
			},
		}
	}
}

func terminalEventType(kind ResultKind) EventType {
	switch kind {
	case AttemptResultExcluded:
		return AttemptExcluded
	case AttemptResultNotSelected:
		return AttemptNotSelected
	default:
		return AttemptFinished
	}
}

// shouldRetry never waits. A suspended callback is not a passing test, and a
// policy skip must not retry even when its reporting cleanup fails.
func (a *attempt) shouldRetry() bool {
	if a.spec.Decision.Kind != DecisionRun {
		return false
	}

	result, ready := a.result()
	if !ready || result.Kind != AttemptResultCompleted {
		return false
	}

	return result.Outcome.Status == AttemptFailed
}
