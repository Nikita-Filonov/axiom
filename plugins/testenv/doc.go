// Package testenv exposes environment variables through typed keys and a lazy
// runner resource. Register Resource with axiom.WithRunnerResources, then read
// keys from resources, hooks, fixtures, plugins, or test actions.
//
// The environment is captured on first access, so configure it before tests
// execute. An explicitly empty variable differs from an unset variable.
package testenv
