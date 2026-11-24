package execution

import "testing"

// Spec describes one attempt and is immutable after submission.
//
// Bind only attaches the native T; it must not run test work or register cleanup.
// Around may forward its continuation at most once. Before, Body, After, and
// Release run only when Decision admits execution. Observe must obey Event's
// observational contract.
type Spec struct {
	Name     string
	Decision Decision
	Parallel bool

	Bind func(*testing.T)

	Around func(func())

	Before  func()
	Body    func()
	After   func()
	Release func()

	Observe func(Event)
}
