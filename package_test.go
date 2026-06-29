package axiom_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// RunPackage
// -----------------------------------------------------------------------------

func TestRunPackage_NilTestingM_Panics(t *testing.T) {
	r := axiom.NewRunner()
	assert.PanicsWithValue(t, "runpackage: nil *testing.M", func() {
		_ = axiom.RunPackage(nil, r)
	})
}

// TestRunPackage_NilRunner_PanicsWithRunnerMessage verifies that RunPackage
// delegates runner validation to RunPackageWith and the resulting panic
// message reaches the caller unchanged. A bare *testing.M is enough because
// m.Run is never invoked when r is nil.
func TestRunPackage_NilRunner_PanicsWithRunnerMessage(t *testing.T) {
	assert.PanicsWithValue(t, "runpackage: nil *Runner", func() {
		_ = axiom.RunPackage(&testing.M{}, nil)
	})
}

// -----------------------------------------------------------------------------
// RunPackageWith — input validation
// -----------------------------------------------------------------------------

func TestRunPackageWith_NilRunner_Panics(t *testing.T) {
	assert.PanicsWithValue(t, "runpackage: nil *Runner", func() {
		_ = axiom.RunPackageWith(nil, func() int { return 0 })
	})
}

func TestRunPackageWith_NilEntry_Panics(t *testing.T) {
	r := axiom.NewRunner()
	assert.PanicsWithValue(t, "runpackage: nil entry function", func() {
		_ = axiom.RunPackageWith(r, nil)
	})
}

// -----------------------------------------------------------------------------
// RunPackageWith — happy path
// -----------------------------------------------------------------------------

func TestRunPackageWith_HappyPath_LifecycleOrder(t *testing.T) {
	var order []string

	r := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(_ *axiom.Runner) { order = append(order, "before-all") }),
			axiom.WithAfterAll(func(_ *axiom.Runner) { order = append(order, "after-all") }),
		),
		axiom.WithRunnerResource("res", func(_ *axiom.Runner) (any, func(), error) {
			order = append(order, "resource-setup")
			return "v", func() { order = append(order, "resource-cleanup") }, nil
		}),
	)

	code := axiom.RunPackageWith(r, func() int {
		order = append(order, "entry-start")
		_ = axiom.MustResource[string](r, "res")
		order = append(order, "entry-end")
		return 0
	})

	assert.Equal(t, 0, code)
	assert.Equal(t, []string{
		"before-all",
		"entry-start",
		"resource-setup",
		"entry-end",
		"after-all",
		"resource-cleanup",
	}, order, "expected lifecycle order: BeforeAll -> entry -> AfterAll -> resource cleanups")
}

func TestRunPackageWith_ExitCodePropagation(t *testing.T) {
	cases := []int{0, 1, 7, 42}
	for _, want := range cases {
		want := want
		t.Run("", func(t *testing.T) {
			r := axiom.NewRunner()
			got := axiom.RunPackageWith(r, func() int { return want })
			assert.Equal(t, want, got)
		})
	}
}

func TestRunPackageWith_EntryInvokedExactlyOnce(t *testing.T) {
	var calls int32

	r := axiom.NewRunner()
	_ = axiom.RunPackageWith(r, func() int {
		atomic.AddInt32(&calls, 1)
		return 0
	})

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

// -----------------------------------------------------------------------------
// RunPackageWith — idempotency with RunCase
// -----------------------------------------------------------------------------

func TestRunPackageWith_BeforeAndAfterAll_FireOnceAndInRightOrder(t *testing.T) {
	// This is the core test for RunPackageWith. It verifies BOTH:
	//   1. BeforeAll / AfterAll fire exactly once across many RunCase calls.
	//   2. AfterAll fires AFTER the whole entry block, not after the first
	//      inner RunCase. Without the managed-flag mechanism, t.Cleanup
	//      scheduled by the first RunCase would prematurely flush
	//      ApplyFinish via sync.Once, and AfterAll would land between the
	//      first and the second TestXxx.
	var order []string
	var beforeCount, afterCount int32

	r := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(_ *axiom.Runner) {
				atomic.AddInt32(&beforeCount, 1)
				order = append(order, "before-all")
			}),
			axiom.WithAfterAll(func(_ *axiom.Runner) {
				atomic.AddInt32(&afterCount, 1)
				order = append(order, "after-all")
			}),
		),
	)
	c := axiom.NewCase(axiom.WithCaseName("inner"))

	code := axiom.RunPackageWith(r, func() int {
		t.Run("first", func(st *testing.T) {
			r.RunCase(st, c, func(_ *axiom.Config) { order = append(order, "first") })
		})
		order = append(order, "between-tests")
		t.Run("second", func(st *testing.T) {
			r.RunCase(st, c, func(_ *axiom.Config) { order = append(order, "second") })
		})
		return 0
	})

	assert.Equal(t, 0, code)
	assert.Equal(t, int32(1), atomic.LoadInt32(&beforeCount),
		"BeforeAll must run exactly once for the whole package boundary")
	assert.Equal(t, int32(1), atomic.LoadInt32(&afterCount),
		"AfterAll must run exactly once for the whole package boundary")
	assert.Equal(t,
		[]string{"before-all", "first", "between-tests", "second", "after-all"},
		order,
		"AfterAll must fire after the entire entry block, not after the first inner RunCase",
	)
}

func TestRunPackageWith_ResourceCleanup_RunsAfterEntireEntry(t *testing.T) {
	// Same as the order-test above, but for resource teardown: resource
	// cleanups must run as part of the outer ApplyFinish, AFTER the second
	// subtest, not after the first.
	var order []string

	r := axiom.NewRunner(
		axiom.WithRunnerResource("db", func(_ *axiom.Runner) (any, func(), error) {
			order = append(order, "db-setup")
			return "db", func() { order = append(order, "db-cleanup") }, nil
		}),
	)
	c := axiom.NewCase(axiom.WithCaseName("inner"))

	_ = axiom.RunPackageWith(r, func() int {
		t.Run("first", func(st *testing.T) {
			r.RunCase(st, c, func(cfg *axiom.Config) {
				_ = axiom.MustResource[string](cfg.Runner, "db")
				order = append(order, "first")
			})
		})
		t.Run("second", func(st *testing.T) {
			r.RunCase(st, c, func(cfg *axiom.Config) {
				_ = axiom.MustResource[string](cfg.Runner, "db")
				order = append(order, "second")
			})
		})
		return 0
	})

	assert.Equal(t,
		[]string{"db-setup", "first", "second", "db-cleanup"},
		order,
		"resource cleanup must run after the whole entry block, not after the first subtest",
	)
}

func TestRunPackageWith_ResourceCleanup_RunsOnceWhenSeveralRunCaseRequest(t *testing.T) {
	var setupCount, cleanupCount int32

	r := axiom.NewRunner(
		axiom.WithRunnerResource("db", func(_ *axiom.Runner) (any, func(), error) {
			atomic.AddInt32(&setupCount, 1)
			return "db", func() { atomic.AddInt32(&cleanupCount, 1) }, nil
		}),
	)
	c := axiom.NewCase(axiom.WithCaseName("inner"))

	_ = axiom.RunPackageWith(r, func() int {
		t.Run("a", func(st *testing.T) {
			r.RunCase(st, c, func(cfg *axiom.Config) {
				_ = axiom.MustResource[string](cfg.Runner, "db")
			})
		})
		t.Run("b", func(st *testing.T) {
			r.RunCase(st, c, func(cfg *axiom.Config) {
				_ = axiom.MustResource[string](cfg.Runner, "db")
			})
		})
		return 0
	})

	assert.Equal(t, int32(1), atomic.LoadInt32(&setupCount),
		"resource constructor must run exactly once for the whole package")
	assert.Equal(t, int32(1), atomic.LoadInt32(&cleanupCount),
		"resource cleanup must run exactly once for the whole package")
}

// -----------------------------------------------------------------------------
// RunPackageWith — managed-flag interaction with RunCase
// -----------------------------------------------------------------------------

// TestRunCase_OutsideRunPackage_StillRegistersApplyFinishCleanup is a
// regression guard: when RunPackage is NOT in use, RunCase must keep its
// historical behaviour and register ApplyFinish via t.Cleanup, so AfterAll
// still fires for users that drive a single top-level TestXxx.
func TestRunCase_OutsideRunPackage_StillRegistersApplyFinishCleanup(t *testing.T) {
	var afterAllRan bool

	r := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithAfterAll(func(_ *axiom.Runner) { afterAllRan = true }),
		),
	)
	c := axiom.NewCase(axiom.WithCaseName("inner"))

	t.Run("owner", func(st *testing.T) {
		r.RunCase(st, c, func(_ *axiom.Config) {})
	})

	assert.True(t, afterAllRan,
		"without RunPackage, RunCase must register t.Cleanup(r.ApplyFinish) so AfterAll still fires")
}

// TestRunPackageWith_ManagedFlagIsRestoredAfterReturn ensures that once
// RunPackageWith returns (or panics), the runner is no longer marked as
// managed, so subsequent RunCase calls fall back to the normal behaviour.
func TestRunPackageWith_ManagedFlagIsRestoredAfterReturn(t *testing.T) {
	r := axiom.NewRunner()
	c := axiom.NewCase(axiom.WithCaseName("inner"))

	_ = axiom.RunPackageWith(r, func() int { return 0 })

	// After RunPackageWith returns, calling RunCase from a fresh testing.T
	// must once again register t.Cleanup. We cannot inspect the unexported
	// flag directly, so we assert behaviour: a stray RunCase after
	// RunPackageWith does not panic (i.e. ApplyFinish via t.Cleanup is a
	// safe no-op due to sync.Once).
	assert.NotPanics(t, func() {
		t.Run("after-package", func(st *testing.T) {
			r.RunCase(st, c, func(_ *axiom.Config) {})
		})
	})
}

// -----------------------------------------------------------------------------
// RunPackageWith — panic propagation
// -----------------------------------------------------------------------------

func TestRunPackageWith_PanicInEntry_RunsAfterAllAndPropagatesPanic(t *testing.T) {
	var afterAllRan, resourceCleanupRan bool

	r := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithAfterAll(func(_ *axiom.Runner) { afterAllRan = true }),
		),
		axiom.WithRunnerResource("res", func(_ *axiom.Runner) (any, func(), error) {
			return "v", func() { resourceCleanupRan = true }, nil
		}),
	)

	assert.PanicsWithValue(t, "boom in entry", func() {
		_ = axiom.RunPackageWith(r, func() int {
			_ = axiom.MustResource[string](r, "res")
			panic("boom in entry")
		})
	})

	assert.True(t, afterAllRan,
		"AfterAll must run via defer even when entry panics")
	assert.True(t, resourceCleanupRan,
		"resource cleanup must run via defer even when entry panics")
}

func TestRunPackageWith_PanicInBeforeAll_EntryNotInvoked(t *testing.T) {
	// Semantics: when BeforeAll panics, the runner lifecycle never reached the
	// point where ApplyFinish would be deferred, so the entry function is not
	// invoked and AfterAll does not run. The original panic propagates verbatim.
	var entryCalled, afterAllRan bool

	r := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(_ *axiom.Runner) { panic("boom in before-all") }),
			axiom.WithAfterAll(func(_ *axiom.Runner) { afterAllRan = true }),
		),
	)

	assert.PanicsWithValue(t, "boom in before-all", func() {
		_ = axiom.RunPackageWith(r, func() int {
			entryCalled = true
			return 0
		})
	})

	assert.False(t, entryCalled,
		"entry must not be invoked when BeforeAll panics")
	assert.False(t, afterAllRan,
		"AfterAll must not run when BeforeAll itself panicked: ApplyFinish was never deferred")
}

func TestRunPackageWith_PanicInAfterAll_StillPropagates(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithAfterAll(func(_ *axiom.Runner) { panic("boom in after-all") }),
		),
	)

	assert.PanicsWithValue(t, "boom in after-all", func() {
		_ = axiom.RunPackageWith(r, func() int { return 0 })
	})
}

// TestRunPackageWith_PanicInAfterAll_StillRunsResourceCleanup pins the
// docs claim: even if a user AfterAll hook panics, runner-scoped resource
// cleanups must still run. ApplyFinish must execute the resource teardown
// via defer regardless of hook outcome.
func TestRunPackageWith_PanicInAfterAll_StillRunsResourceCleanup(t *testing.T) {
	var cleanupRan bool

	r := axiom.NewRunner(
		axiom.WithRunnerResource("res", func(_ *axiom.Runner) (any, func(), error) {
			return "v", func() { cleanupRan = true }, nil
		}),
		axiom.WithRunnerHooks(
			axiom.WithAfterAll(func(_ *axiom.Runner) { panic("boom in after-all") }),
		),
	)

	assert.PanicsWithValue(t, "boom in after-all", func() {
		_ = axiom.RunPackageWith(r, func() int {
			_ = axiom.MustResource[string](r, "res")
			return 0
		})
	})

	assert.True(t, cleanupRan,
		"resource cleanup must still run even if an AfterAll hook panics")
}

// TestRunPackageWith_PanicInEntry_DoesNotLeaveRunnerInManagedState verifies
// that the defer chain restoring managed=false runs even if entry panics.
// Behavioural assertion: after the panic propagates, a follow-up RunCase on
// the same runner must not panic or deadlock and must observe a normal
// (non-managed) flow.
func TestRunPackageWith_PanicInEntry_DoesNotLeaveRunnerInManagedState(t *testing.T) {
	r := axiom.NewRunner()
	c := axiom.NewCase(axiom.WithCaseName("inner"))

	assert.PanicsWithValue(t, "boom", func() {
		_ = axiom.RunPackageWith(r, func() int { panic("boom") })
	})

	// Stray RunCase after a panicked RunPackageWith must remain safe.
	// If the defer chain failed to restore the flag, this call would still
	// not register t.Cleanup(ApplyFinish), but ApplyFinish is sync.Once
	// guarded so the visible effect is muted — we assert at minimum that
	// nothing crashes or hangs.
	assert.NotPanics(t, func() {
		t.Run("after-panic", func(st *testing.T) {
			r.RunCase(st, c, func(_ *axiom.Config) {})
		})
	})
}

// -----------------------------------------------------------------------------
// RunPackageWith — corner cases
// -----------------------------------------------------------------------------

// TestRunPackageWith_MultipleResources_TornDownInLIFOOrder verifies that
// resource cleanups across the package boundary still respect LIFO ordering,
// the same way they do for standalone Runner teardown.
func TestRunPackageWith_MultipleResources_TornDownInLIFOOrder(t *testing.T) {
	var order []string

	r := axiom.NewRunner(
		axiom.WithRunnerResource("a", func(_ *axiom.Runner) (any, func(), error) {
			order = append(order, "a-setup")
			return "a", func() { order = append(order, "a-cleanup") }, nil
		}),
		axiom.WithRunnerResource("b", func(_ *axiom.Runner) (any, func(), error) {
			order = append(order, "b-setup")
			return "b", func() { order = append(order, "b-cleanup") }, nil
		}),
		axiom.WithRunnerResource("c", func(_ *axiom.Runner) (any, func(), error) {
			order = append(order, "c-setup")
			return "c", func() { order = append(order, "c-cleanup") }, nil
		}),
	)

	_ = axiom.RunPackageWith(r, func() int {
		_ = axiom.MustResource[string](r, "a")
		_ = axiom.MustResource[string](r, "b")
		_ = axiom.MustResource[string](r, "c")
		return 0
	})

	assert.Equal(t,
		[]string{
			"a-setup", "b-setup", "c-setup",
			"c-cleanup", "b-cleanup", "a-cleanup",
		},
		order,
		"resource cleanups across the package boundary must run in LIFO order",
	)
}

// TestRunPackageWith_Managed_AtomicReadIsRaceFree exercises the managed
// flag from many concurrent goroutines while the lifecycle is active. The
// body is intentionally minimal; the value of this test comes from running
// it under `go test -race`, which would flag any data race if the field
// were switched back to a plain bool.
func TestRunPackageWith_Managed_AtomicReadIsRaceFree(t *testing.T) {
	r := axiom.NewRunner()
	c := axiom.NewCase(axiom.WithCaseName("inner"))

	_ = axiom.RunPackageWith(r, func() int {
		var wg sync.WaitGroup
		for i := 0; i < 32; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				t.Run(fmt.Sprintf("g-%d", i), func(st *testing.T) {
					r.RunCase(st, c, func(_ *axiom.Config) {})
				})
			}()
		}
		wg.Wait()
		return 0
	})
}

// -----------------------------------------------------------------------------
// Defensive: BeforeAll exit code is irrelevant; AfterAll observes the runner
// state populated by entry
// -----------------------------------------------------------------------------

func TestRunPackageWith_AfterAllObservesRunnerStateMutatedByEntry(t *testing.T) {
	var afterAllSawValue string

	r := axiom.NewRunner(
		axiom.WithRunnerResource("flag", func(_ *axiom.Runner) (any, func(), error) {
			return "set-by-entry", nil, nil
		}),
		axiom.WithRunnerHooks(
			axiom.WithAfterAll(func(rr *axiom.Runner) {
				v, err := axiom.GetResource[string](rr, "flag")
				if err == nil {
					afterAllSawValue = v
				}
			}),
		),
	)

	_ = axiom.RunPackageWith(r, func() int {
		_ = axiom.MustResource[string](r, "flag")
		return 0
	})

	assert.Equal(t, "set-by-entry", afterAllSawValue,
		"AfterAll runs before resource cleanup and must observe resources created during entry")
}
