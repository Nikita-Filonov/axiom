package axiom

import (
	"testing"
)

// Config is the merged execution state for one test attempt. Retry attempts get
// separate Config values, including separate fixture caches and local state.
// Plugins and the test action receive the Config for the current attempt.
// Meta, Skip, Retry, Hooks, Context, Runtime, Parallel, and Fixtures contain
// merged settings; Local holds state created during this attempt.
type Config struct {
	// RootT is the original *testing.T passed to the runner.
	RootT *testing.T
	// SubT is the *testing.T for the current subtest, when one is active.
	SubT *testing.T

	// Runner is the shared runner that owns this attempt.
	Runner *Runner
	// Case is the attempt's copy of the declarative case.
	Case *Case

	Meta     Meta
	Skip     Skip
	Retry    Retry
	Local    Local
	Hooks    Hooks
	Context  Context
	Runtime  Runtime
	Parallel Parallel
	Fixtures Fixtures
}

type testLifecyclePanic struct {
	value any
}

// T returns the active subtest's *testing.T, or RootT when no subtest is active.
func (c *Config) T() *testing.T {
	if c.SubT != nil {
		return c.SubT
	}

	return c.RootT
}

// Log emits l as an Event and dispatches it to runtime log sinks.
func (c *Config) Log(l Log) {
	c.Event(NewLogEvent(l))
	c.Runtime.Log(l)
}

// Step executes fn as a named step through hooks and runtime wrappers, emitting
// start, finish, and panic facts as appropriate.
func (c *Config) Step(name string, fn func()) {
	c.Event(NewEvent(EventTypeStepStart, WithEventName(name)))
	defer func() {
		if r := recover(); r != nil {
			c.Event(NewEvent(EventTypeStepPanic, WithEventName(name), WithEventMessage(r)))
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in step %q: %v", name, r)
			}
		}

		c.Hooks.ApplyAfterStep(c, name)
		c.Event(NewEvent(EventTypeStepFinish, WithEventName(name)))
	}()

	c.Hooks.ApplyBeforeStep(c, name)
	c.Runtime.Step(name, fn)
}

// Test runs one attempt through runtime test wraps, hooks, and fixture cleanup.
// AfterTest hooks run before fixture cleanups, which run in reverse setup order.
func (c *Config) Test(action TestAction) {
	c.Event(NewEvent(EventTypeCaseStart))
	defer func() {
		if r := recover(); r != nil {
			if lifecyclePanic, ok := r.(testLifecyclePanic); ok {
				panic(lifecyclePanic.value)
			}

			c.Event(NewEvent(EventTypeCasePanic, WithEventMessage(r)))
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in test %q: %v", c.Case.Name, r)
			}
		}

		c.Event(NewEvent(EventTypeCaseFinish))
	}()

	c.Runtime.Test(c, func(current *Config) {
		defer func() {
			defer func() {
				if r := recover(); r != nil {
					panic(testLifecyclePanic{value: r})
				}
			}()
			defer current.Fixtures.Teardown(current)
			current.Hooks.ApplyAfterTest(current)
		}()

		current.Hooks.ApplyBeforeTest(current)
		action(current)
	})
}

// Event dispatches e to runtime event sinks.
func (c *Config) Event(e Event) { c.Runtime.Event(e) }

// Setup executes fn immediately as a named setup operation through runtime
// wrappers. It does not schedule fn for later execution.
func (c *Config) Setup(name string, fn func()) {
	c.Event(NewEvent(EventTypeSetupStart, WithEventName(name)))
	defer func() {
		if r := recover(); r != nil {
			c.Event(NewEvent(EventTypeSetupPanic, WithEventName(name), WithEventMessage(r)))
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in setup %q: %v", name, r)
			}
		}

		c.Event(NewEvent(EventTypeSetupFinish, WithEventName(name)))
	}()

	c.Runtime.Setup(name, fn)
}

// Teardown executes fn immediately as a named teardown operation through
// runtime wrappers. It does not schedule fn for end-of-attempt cleanup.
func (c *Config) Teardown(name string, fn func()) {
	c.Event(NewEvent(EventTypeTeardownStart, WithEventName(name)))
	defer func() {
		if r := recover(); r != nil {
			c.Event(NewEvent(EventTypeTeardownPanic, WithEventName(name), WithEventMessage(r)))
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in teardown %q: %v", name, r)
			}
		}

		c.Event(NewEvent(EventTypeTeardownFinish, WithEventName(name)))
	}()

	c.Runtime.Teardown(name, fn)
}

// Assert emits a as an Event and dispatches it to runtime assert sinks. Axiom
// does not evaluate the assertion or fail the test on its own.
func (c *Config) Assert(a Assert) {
	c.Event(NewAssertEvent(a))
	c.Runtime.Assert(a)
}

// Artefact emits a as an Event and dispatches it to runtime artefact sinks.
func (c *Config) Artefact(a Artefact) {
	c.Event(NewArtefactEvent(a))
	c.Runtime.Artefact(a)
}

// ApplyPlugins runs Runner plugins followed by Case plugins on c.
func (c *Config) ApplyPlugins() {
	for _, p := range c.Runner.Plugins {
		p(c)
	}
	for _, p := range c.Case.Plugins {
		p(c)
	}
}

func (c *Config) applySkipPolicy() {
	if c.Skip.Enabled {
		c.T().Skip(c.Skip.Reason)
	}
}

func (c *Config) applyParallelPolicy() {
	if c.Parallel.Enabled {
		c.T().Parallel()
	}
}
