package testleaks

import (
	"context"
	"crypto/rand"
	"runtime/pprof"
	"strconv"

	"github.com/Nikita-Filonov/axiom"
)

func wrapAttempt(state *attemptState, c Config) axiom.WrapTestAction {
	return func(next axiom.TestAction) axiom.TestAction {
		return func(current *axiom.Config) {
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
	if cfg.Execution.ID != "" {
		return cfg.Execution.ID + "/" + strconv.Itoa(cfg.Execution.Attempt)
	}
	return rand.Text()
}
