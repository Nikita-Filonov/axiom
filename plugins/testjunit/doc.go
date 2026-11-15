// Package testjunit exports finished Axiom case attempts as JUnit XML.
// Its recorder collects outcomes after test cleanup and reconciles late
// failures in the parent test's cleanup.
// WithOutputFile publishes incremental XML snapshots without TestMain.
// WithReporter retains access to explicit Write and WriteFile exports.
package testjunit
