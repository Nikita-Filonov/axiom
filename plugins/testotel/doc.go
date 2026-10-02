// Package testotel turns Axiom case attempts into OpenTelemetry spans. It uses
// the caller's tracer provider and leaves SDK setup, export, and shutdown to
// the caller. Use Context to propagate the active attempt span to operations
// performed by a test.
package testotel
