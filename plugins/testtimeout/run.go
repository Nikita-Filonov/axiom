package testtimeout

import (
	"time"

	"github.com/Nikita-Filonov/axiom"
)

func runWithTimeout(c *axiom.Config, cfg Config, next axiom.TestAction) {
	deadline := time.Now().Add(cfg.Timeout)
	if cfg.ContextDeadline {
		cancel := applyContextDeadline(c, deadline)
		defer cancel()
	}

	type result struct {
		panicValue    any
		panicOccurred bool
		finishedAt    time.Time
	}
	done := make(chan result, 1)

	go func() {
		outcome := result{}
		defer func() {
			outcome.finishedAt = time.Now()
			done <- outcome
		}()
		defer func() {
			if r := recover(); r != nil {
				outcome.panicValue, outcome.panicOccurred = r, true
			}
		}()

		next(c)
	}()

	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()

	var outcome result
	select {
	case outcome = <-done:
	case <-timer.C:
		// Both channels may be ready; a body that finished before the deadline wins.
		select {
		case outcome = <-done:
		default:
			reportTimeout(c, cfg)
			return
		}
	}

	// A body returning because its context expired must still fail the attempt.
	if !outcome.finishedAt.Before(deadline) {
		reportTimeout(c, cfg)
		return
	}
	if outcome.panicOccurred {
		panic(outcome.panicValue)
	}
}
