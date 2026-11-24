// Package execution schedules native Go subtests and observes their lifecycle.
// It depends only on testing and other standard library packages; declarations,
// configuration merging, metadata, plugins, and reporting belong to the caller.
//
// Run accepts a Plan that prepares a fresh Spec for each attempt. Each spec has an
// immutable Decision: run, skip, or exclude. A skip enters Around for reporting
// but does not enter Before, Body, After, or Release. An exclusion enters none
// of these callbacks and has no test outcome. Once a running attempt enters
// Before, deferred After and Release execute even on panic or Goexit.
//
// Callback exit and native completion are distinct. A test may have unfinished
// children or cleanup after its callbacks unwind. Run observes the final native
// status only after completion, with failure taking precedence over skip. A
// parallel test can outlive Run; Complete receives its final result at a safe
// parent boundary. Callers must not wait for it while holding its parent open.
//
// Observe receives non-terminal lifecycle facts; Complete receives exactly one
// terminal Result. Both callbacks receive snapshots, must be concurrency-safe,
// and must not perform test actions. The caller owns correlation and any public
// event translation. The engine does not replace testing.T or intercept
// arbitrary calls to it.
package execution
