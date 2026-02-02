package axiom

import (
	"testing"
)

type Config struct {
	RootT *testing.T
	SubT  *testing.T

	Runner *Runner
	Case   *Case

	ID       string
	Name     string
	Meta     Meta
	Skip     Skip
	Retry    Retry
	Hooks    Hooks
	Params   any
	Context  Context
	Runtime  Runtime
	Parallel Parallel
	Fixtures Fixtures
}

func (c *Config) Log(l Log) { c.Runtime.Log(l) }

func (c *Config) Step(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in step %q: %v", name, r)
			}
		}

		c.Hooks.ApplyAfterStep(c, name)
	}()

	c.Hooks.ApplyBeforeStep(c, name)
	c.Runtime.Step(name, fn)
}

func (c *Config) Test(action TestAction) {
	defer func() {
		if r := recover(); r != nil {
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in test %q: %v", c.Name, r)
			}
		}

		c.Hooks.ApplyAfterTest(c)
	}()

	c.Hooks.ApplyBeforeTest(c)
	c.Runtime.Test(c, action)
}

func (c *Config) Setup(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in setup %q: %v", name, r)
			}
		}
	}()

	c.Runtime.Setup(name, fn)
}

func (c *Config) Teardown(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if c.SubT != nil {
				c.SubT.Helper()
				c.SubT.Errorf("panic in teardown %q: %v", name, r)
			}
		}
	}()

	c.Runtime.Teardown(name, fn)
}

func (c *Config) Assert(a Assert) { c.Runtime.Assert(a) }

func (c *Config) Artefact(a Artefact) { c.Runtime.Artefact(a) }

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
