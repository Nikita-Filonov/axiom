package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttemptExecutePanicPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    bool
		after   bool
		release bool
	}{
		{name: "normal"},
		{
			name: "body",
			body: true,
		},
		{
			name:  "after",
			after: true,
		},
		{
			name:    "release",
			release: true,
		},
		{
			name:  "body then after",
			body:  true,
			after: true,
		},
		{
			name:    "body then release",
			body:    true,
			release: true,
		},
		{
			name:    "after then release",
			after:   true,
			release: true,
		},
		{
			name:    "body then after then release",
			body:    true,
			after:   true,
			release: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type panicValue struct{ phase string }
			bodyPanic := &panicValue{phase: "body"}
			afterPanic := &panicValue{phase: "after"}
			releasePanic := &panicValue{phase: "release"}
			var order []string
			var events []EventType
			native := &testing.T{}
			a := &attempt{
				spec: Spec{
					Decision: Decision{Kind: DecisionRun},
					Around: func(next func()) {
						defer func() { order = append(order, "around exit") }()
						next()
					},
					Body: func() {
						order = append(order, "body")
						if tc.body {
							panic(bodyPanic)
						}
					},
					After: func() {
						order = append(order, "after")
						if tc.after {
							panic(afterPanic)
						}
					},
					Release: func() {
						order = append(order, "release")
						if tc.release {
							panic(releasePanic)
						}
					},
					Observe: func(event Event) { events = append(events, event.Type) },
				},
			}

			var recovered any
			func() {
				defer func() { recovered = recover() }()
				a.execute(native)
			}()

			expectedEvents := []EventType{AttemptStarted}
			switch {
			case tc.release:
				assert.Same(t, releasePanic, recovered)
			case tc.after:
				assert.Same(t, afterPanic, recovered)
			default:
				assert.Nil(t, recovered)
				if tc.body {
					expectedEvents = append(expectedEvents, AttemptPanicRecovered)
				}
				expectedEvents = append(expectedEvents, AttemptExecutionFinished)
			}
			assert.Equal(t, tc.body && !tc.after && !tc.release, native.Failed(), "only a recovered panic is converted to a failure here")
			assert.Equal(t, []string{"body", "after", "release", "around exit"}, order)
			assert.Equal(t, expectedEvents, events)
		})
	}
}

func TestRunParallelRetryUsesOneContainer(t *testing.T) {
	var names []string
	var prepared []int

	t.Run("parent", func(parent *testing.T) {
		Run(parent, Plan{
			Name:        "case",
			Parallel:    true,
			MaxAttempts: 2,

			Prepare: func(ctx PrepareContext) Spec {
				prepared = append(prepared, ctx.Number)

				return Spec{
					Name:     "attempt",
					Parallel: true,
					Decision: Decision{Kind: DecisionRun},
					Bind:     func(t *testing.T) { names = append(names, t.Name()) },
				}
			},
		})

		assert.Empty(parent, names, "the container has not resumed")
	})

	assert.Equal(t, []int{1}, prepared, "a passing attempt stops retries")
	assert.Equal(t, []string{t.Name() + "/parent/case/attempt"}, names)
}
