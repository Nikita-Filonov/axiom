package testleaks

import (
	"context"
	"crypto/rand"
	"runtime/pprof"
	"strconv"

	"github.com/Nikita-Filonov/axiom"
)

const labelKey = "github.com/Nikita-Filonov/axiom/plugins/testleaks.attempt"

func wrapAttempt(state *attemptState, c Config) axiom.WrapTestAction {
	return func(next axiom.TestAction) axiom.TestAction {
		return func(current *axiom.Config) {
			if current.Attempt.Decision.Kind != axiom.DecisionRun {
				next(current)
				return
			}
			t := current.T()
			if t == nil {
				next(current)
				return
			}

			id := attemptID(current)
			t.Cleanup(func() { verify(t, state, id, c, findGoroutines) })
			parent := current.Context.Raw
			if parent == nil {
				parent = context.Background()
			}
			pprof.Do(parent, pprof.Labels(labelKey, id), func(context.Context) {
				next(current)
			})
		}
	}
}

func attemptID(cfg *axiom.Config) string {
	if cfg.Attempt.RunID != "" {
		return cfg.Attempt.RunID + "/" + strconv.Itoa(cfg.Attempt.Number)
	}
	return rand.Text()
}
