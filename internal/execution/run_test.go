package execution

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the engine with plain callbacks, without a Runner or
// Config. Public integration tests cover the separate Axiom adapter.
func TestEngineAttemptLifecycle(t *testing.T) {
	var calls []string
	var events []Event
	var completed []Result
	var native *testing.T
	a := &attempt{
		spec: Spec{
			Name:     "attempt",
			Decision: Decision{Kind: DecisionRun},
			Bind:     func(t *testing.T) { native = t },
			Around: func(next func()) {
				calls = append(calls, "enter")
				defer func() { calls = append(calls, "exit") }()
				next()
			},
			Before: func() { calls = append(calls, "before") },
			Body: func() {
				calls = append(calls, "body")
				native.Cleanup(func() { calls = append(calls, "native cleanup") })
			},
			After:    func() { calls = append(calls, "after") },
			Release:  func() { calls = append(calls, "release") },
			Observe:  func(e Event) { events = append(events, e) },
			Complete: func(r Result) { completed = append(completed, r) },
		},
	}
	a.run(t)

	assert.Equal(t, []string{"enter", "before", "body", "after", "release", "exit", "native cleanup"}, calls)

	result, ready := a.result()
	require.True(t, ready)
	require.NotNil(t, result.Outcome)
	assert.Equal(t, AttemptPassed, result.Outcome.Status)
	assert.Equal(t, t.Name()+"/attempt", result.Outcome.TestName)
	assert.False(t, result.Outcome.CleanupAt.Before(result.Outcome.StartedAt))
	assert.Equal(t, AttemptStarted, events[0].Type)
	assert.Equal(t, AttemptExecutionFinished, events[len(events)-1].Type)
	require.Len(t, completed, 1)
	assert.Equal(t, AttemptPassed, completed[0].Outcome.Status)

	// A reader cannot mutate engine state, and late work cannot reopen an attempt.
	result.Outcome.Status = AttemptFailed
	a.emit(Event{Type: AttemptStarted})
	a.finalize()
	again, _ := a.result()
	assert.Equal(t, AttemptPassed, again.Outcome.Status)
	assert.Len(t, events, 2)
	assert.Len(t, completed, 1)
}

func TestEngineSelectionBranches(t *testing.T) {
	for _, kind := range []DecisionKind{DecisionSkip, DecisionExclude} {
		t.Run(string(kind), func(t *testing.T) {
			var wrapped, cleaned bool
			var events []Event
			var completed []Result
			var native *testing.T
			unexpected := func() { t.Error("execution callback ran") }
			a := &attempt{
				spec: Spec{
					Name: "selected",
					Decision: Decision{
						Kind:   kind,
						Reason: "disabled",
					},
					Parallel: true,
					Bind:     func(t *testing.T) { native = t },
					Around: func(next func()) {
						wrapped = true
						native.Cleanup(func() { cleaned = true })
						next()
					},
					Before:   unexpected,
					Body:     unexpected,
					After:    unexpected,
					Release:  unexpected,
					Observe:  func(e Event) { events = append(events, e) },
					Complete: func(r Result) { completed = append(completed, r) },
				},
			}
			a.run(t)

			result, ready := a.result()
			require.True(t, ready, "terminal policy must not suspend in Parallel")
			if kind == DecisionExclude {
				assert.False(t, wrapped)
				assert.False(t, cleaned)
				assert.Equal(t, ResultExcluded, result.Kind)
				assert.Nil(t, result.Outcome)
				assert.Empty(t, events)
			} else {
				assert.True(t, wrapped)
				assert.True(t, cleaned)
				assert.Equal(t, AttemptSkipped, result.Outcome.Status)
				require.Len(t, events, 2)
				assert.Equal(t, AttemptStarted, events[0].Type)
				assert.Equal(t, AttemptExecutionFinished, events[1].Type)
			}
			require.Len(t, completed, 1)
			assert.Equal(t, result, completed[0])
		})
	}
}

func TestEngineParallelCompletion(t *testing.T) {
	var a *attempt
	var events []Event
	var completed []Result
	var cleaned bool
	t.Run("parent", func(parent *testing.T) {
		var native *testing.T
		a = &attempt{
			spec: Spec{
				Name:     "parallel",
				Decision: Decision{Kind: DecisionRun},
				Parallel: true,
				Bind:     func(t *testing.T) { native = t },
				Body:     func() { native.Cleanup(func() { cleaned = true }) },
				Observe:  func(e Event) { events = append(events, e) },
				Complete: func(r Result) { completed = append(completed, r) },
			},
		}
		a.run(parent)

		_, ready := a.result()
		assert.False(parent, ready)
		assert.False(parent, cleaned)
		assert.Empty(parent, events, "a paused attempt has not started execution")
		assert.Empty(parent, completed)
	})

	result, ready := a.result()
	require.True(t, ready)
	assert.True(t, cleaned)
	assert.Equal(t, AttemptPassed, result.Outcome.Status)
	assert.Equal(t, AttemptStarted, events[0].Type)
	assert.Equal(t, AttemptExecutionFinished, events[len(events)-1].Type)
	require.Len(t, completed, 1)
	assert.Equal(t, result, completed[0])
}

func TestEngineNativeParallelIsNotAnEarlyPass(t *testing.T) {
	var a *attempt
	t.Run("parent", func(t *testing.T) {
		var native *testing.T
		a = &attempt{
			spec: Spec{
				Name:     "direct",
				Decision: Decision{Kind: DecisionRun},
				Bind:     func(t *testing.T) { native = t },
				Body:     func() { native.Parallel() },
			},
		}
		a.run(t)

		_, ready := a.result()
		assert.False(t, ready)
	})

	result, ready := a.result()
	require.True(t, ready)
	assert.Equal(t, AttemptPassed, result.Outcome.Status)
}

func TestEngineOutcomePrecedenceAndContinuationGuard(t *testing.T) {
	failed := &testing.T{}
	failed.Fail()
	a := &attempt{
		t:    failed,
		spec: Spec{Decision: Decision{Kind: DecisionRun}},
	}
	a.finalize()

	result, ready := a.result()
	require.True(t, ready)
	assert.Equal(t, AttemptFailed, result.Outcome.Status)

	var events []Event
	a = &attempt{spec: Spec{
		Decision: Decision{Kind: DecisionRun},
		Around: func(next func()) {
			next()
			next()
		},
		Observe: func(e Event) { events = append(events, e) },
	}}
	a.execute(failed)
	assert.Equal(t, AttemptPanicRecovered, events[len(events)-2].Type)
	assert.Contains(t, events[len(events)-2].Message, "more than once")
}

func TestEnginePreparesRetriesLazily(t *testing.T) {
	var prepared []int
	plan := Plan{
		MaxAttempts: 2,
		Prepare: func(ctx PrepareContext) Spec {
			prepared = append(prepared, ctx.Number)

			return Spec{Decision: Decision{Kind: DecisionRun}}
		},
	}

	Run(t, plan)
	assert.Equal(t, []int{1}, prepared)
}

func TestEngineNativeOutcomes(t *testing.T) {
	for _, mode := range []string{"cleanup", "cleanup-panic", "child", "runtime-skip", "fail-skip", "policy-skip-fail", "panic", "retry", "filter", "concurrent"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.json")
			pattern := "^TestEngineProbe$"
			if mode == "filter" {
				pattern += "/selected$"
			}
			args := []string{"-test.run=" + pattern}
			// Share Go's coverage directory so subprocess-only failure paths
			// contribute to the same package coverage report.
			if f := flag.Lookup("test.gocoverdir"); f != nil && f.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+f.Value.String())
			}

			cmd := exec.Command(os.Args[0], args...)
			cmd.Env = append(os.Environ(), "AXIOM_ENGINE_PROBE="+mode, "AXIOM_ENGINE_OUTPUT="+path)

			output, err := cmd.CombinedOutput()
			if mode == "runtime-skip" || mode == "filter" || mode == "concurrent" {
				require.NoError(t, err, "%s", output)
			} else {
				require.Error(t, err, "%s", output)
			}

			data, err := os.ReadFile(path)
			require.NoError(t, err, "%s", output)
			var snapshot engineProbeSnapshot
			require.NoError(t, json.Unmarshal(data, &snapshot))
			require.NotEmpty(t, snapshot.Results)

			last := snapshot.Results[len(snapshot.Results)-1]
			if mode == "filter" {
				assert.Equal(t, ResultNotSelected, last.Kind)
				assert.Nil(t, last.Outcome)
				return
			}
			assert.Equal(t, ResultCompleted, last.Kind)
			status := AttemptFailed
			if mode == "runtime-skip" || mode == "retry" {
				status = AttemptSkipped
			} else if mode == "concurrent" {
				status = AttemptPassed
			}
			assert.Equal(t, status, last.Outcome.Status)
			if mode == "policy-skip-fail" {
				assert.Len(t, snapshot.Results, 1, "a failed policy skip must not retry")
			}
			if mode == "retry" {
				require.Len(t, snapshot.Results, 2)
				assert.Equal(t, AttemptFailed, snapshot.Results[0].Outcome.Status)
				assert.GreaterOrEqual(t, snapshot.Results[1].Outcome.StartedAt.Sub(snapshot.Results[0].Outcome.CleanupAt), time.Millisecond)
			}
		})
	}
}

type engineProbeSnapshot struct {
	Events  []Event
	Results []Result
}

func TestEngineProbe(t *testing.T) {
	mode := os.Getenv("AXIOM_ENGINE_PROBE")
	if mode == "" {
		t.Skip("subprocess only")
	}
	var mu sync.Mutex
	var snapshot engineProbeSnapshot
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		data, err := json.Marshal(snapshot)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(os.Getenv("AXIOM_ENGINE_OUTPUT"), data, 0600))
	})

	prepare := func(ctx PrepareContext) Spec {
		var native *testing.T
		spec := Spec{
			Name:     "other",
			Decision: Decision{Kind: DecisionRun},
			Bind:     func(t *testing.T) { native = t },
			Observe: func(e Event) {
				mu.Lock()
				snapshot.Events = append(snapshot.Events, e)
				mu.Unlock()
			},
			Complete: func(r Result) {
				mu.Lock()
				snapshot.Results = append(snapshot.Results, r)
				mu.Unlock()
			},
			Body: func() {
				switch mode {
				case "cleanup":
					native.Cleanup(func() { native.Error("cleanup failure") })
				case "cleanup-panic":
					native.Cleanup(func() { panic("cleanup panic") })
				case "child":
					native.Run("child", func(t *testing.T) {
						t.Parallel()
						t.Error("child failure")
					})
				case "runtime-skip":
					native.Skip("runtime reason")
				case "fail-skip":
					native.Error("failure wins")
					native.SkipNow()
				case "panic":
					panic("body panic")
				case "retry":
					native.Error("first failure")
				case "concurrent":
					// Observer and result reads stay safe for concurrent cases.
					native.Log("concurrent body")
				}
			},
		}
		if mode == "retry" && ctx.Number > 1 {
			spec.Decision = Decision{
				Kind:   DecisionSkip,
				Reason: "disabled after failure",
			}
		}
		if mode == "policy-skip-fail" {
			spec.Decision = Decision{
				Kind:   DecisionSkip,
				Reason: "disabled",
			}
			spec.Around = func(next func()) {
				native.Cleanup(func() { native.Error("reporting cleanup failure") })
				next()
			}
		}
		return spec
	}

	if mode == "concurrent" {
		for i := range 10 {
			spec := prepare(PrepareContext{Number: i + 1})
			spec.Parallel = true
			a := &attempt{spec: spec}
			a.run(t)
		}
		return
	}
	Run(t, Plan{
		Name:        "run",
		MaxAttempts: 3,
		RetryDelay:  time.Millisecond,
		Prepare:     prepare,
	})
}
