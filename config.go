package axiom

import (
	"testing"
)

type Config struct {
	RootT *testing.T
	SubT  *testing.T

	Runner *Runner
	Case   *Case

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

func (c *Config) T() *testing.T {
	if c.SubT != nil {
		return c.SubT
	}

	return c.RootT
}

func (c *Config) Log(l Log) {
	c.Event(NewLogEvent(l))
	c.Runtime.Log(l)
}

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

func (c *Config) Test(action TestAction) {
	c.Event(NewEvent(EventTypeCaseStart))
	defer func() {
		if r := recover(); r != nil {
			c.Event(NewEvent(EventTypeCasePanic, WithEventMessage(r)))
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in test %q: %v", c.Case.Name, r)
			}
		}

		c.Fixtures.Teardown(c)
		c.Hooks.ApplyAfterTest(c)
		c.Event(NewEvent(EventTypeCaseFinish))
	}()

	c.Hooks.ApplyBeforeTest(c)
	c.Runtime.Test(c, action)
}

func (c *Config) Event(e Event) { c.Runtime.Event(e) }

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

func (c *Config) Assert(a Assert) {
	c.Event(NewAssertEvent(a))
	c.Runtime.Assert(a)
}

func (c *Config) Artefact(a Artefact) {
	c.Event(NewArtefactEvent(a))
	c.Runtime.Artefact(a)
}

func (c *Config) ApplyPlugins() {
	for _, p := range c.Runner.Plugins {
		p(c)
	}
	for _, p := range c.Case.Plugins {
		p(c)
	}
}

func (c *Config) ApplyExecutionPolicy() {
	t := c.RootT
	if c.SubT != nil {
		t = c.SubT
	}

	if c.Skip.Enabled {
		t.Skip(c.Skip.Reason)
	}

	if c.Parallel.Enabled {
		t.Parallel()
	}
}
