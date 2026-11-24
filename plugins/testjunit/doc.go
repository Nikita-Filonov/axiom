// Package testjunit exports finished Axiom case attempts as JUnit XML.
// Its recorder publishes during test cleanup, then confirms status and timing
// through Axiom final attempt results without appending a second record.
// WithOutputFile publishes incremental XML snapshots without TestMain.
// WithReporter retains access to explicit Write and WriteFile exports.
package testjunit
