// Package axiom adds configurable test execution to Go's testing package.
// A [Runner] combines shared settings with a declarative [Case] and passes a
// [Config] to the test action for each execution attempt. [Fixture] values live
// for one attempt; [Resource] values are shared for the runner's lifetime.
// Tests remain ordinary Go tests discovered and run by go test.
package axiom
