// Package teststats records the outcome of every Axiom case attempt. Attempts
// of one RunCase invocation share a run ID, so retries are grouped into runs
// whose derived status distinguishes flaky cases from failed ones.
package teststats
