// Package teststats records the outcome of every Axiom case attempt. Attempts
// of one RunCase invocation share a run ID, so retries are grouped into runs
// whose derived status distinguishes flaky cases from failed ones.
// Plugin accepts options: WithStats supplies a queryable collector, and
// WithOutputDir enables versioned JSON snapshots during test execution.
package teststats
