package axiom

import (
	"testing"
)

type TestAction func(cfg *Config)
type StepAction func()

type WrapTestAction func(next TestAction) TestAction
type WrapStepAction func(name string, next StepAction) StepAction

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
	Parallel Parallel
	Fixtures Fixtures

	TestWraps []WrapTestAction
	StepWraps []WrapStepAction
}

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

	wrapped := fn
	for i := len(c.StepWraps) - 1; i >= 0; i-- {
		wrapped = c.StepWraps[i](name, wrapped)
	}

	wrapped()
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

	wrapped := action
	for i := len(c.TestWraps) - 1; i >= 0; i-- {
		wrapped = c.TestWraps[i](wrapped)
	}

	wrapped(c)
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
