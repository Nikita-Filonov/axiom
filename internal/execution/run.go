package execution

import (
	"testing"
	"time"
)

// Run schedules a logical run and preserves ordinary Go subtest names. Only
// parallel retries need a container: parallelize it once and await its attempts.
// Run can return while a parallel test is suspended. Final results are delivered
// to Spec.Complete after native completion, never inferred from an early return.
func Run(parent *testing.T, plan Plan) {
	r := runner{plan: plan}
	r.run(parent)
}

// runner owns scheduling policy. The parent remains explicit because parallel
// retries execute inside a native container rather than the original parent.
type runner struct {
	plan Plan
}

func (r runner) run(parent *testing.T) {
	first := r.plan.Prepare(PrepareContext{Number: 1})

	if r.needsContainer(first) {
		r.runContainer(parent, first)
		return
	}

	r.runAttempts(parent, first, false)
}

func (r runner) needsContainer(first Spec) bool {
	return r.plan.Parallel &&
		r.plan.MaxAttempts > 1 &&
		first.Decision.Kind == DecisionRun
}

func (r runner) runContainer(parent *testing.T, first Spec) {
	parent.Run(r.plan.Name, func(t *testing.T) {
		t.Parallel()
		r.runAttempts(t, first, true)
	})
}

func (r runner) runAttempts(parent *testing.T, spec Spec, sequential bool) {
	for number := 1; ; number++ {
		if sequential {
			spec.Parallel = false
		}

		a := &attempt{spec: spec}
		a.run(parent)

		if number >= r.plan.MaxAttempts || !a.shouldRetry() {
			return
		}

		spec = r.next(number + 1)
	}
}

func (r runner) next(number int) Spec {
	if r.plan.RetryDelay > 0 {
		time.Sleep(r.plan.RetryDelay)
	}

	return r.plan.Prepare(PrepareContext{Number: number})
}
