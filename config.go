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
	c.Hooks.ApplyBeforeStep(c, name)

	wrapped := fn
	for i := len(c.StepWraps) - 1; i >= 0; i-- {
		wrapped = c.StepWraps[i](name, wrapped)
	}

	wrapped()

	c.Hooks.ApplyAfterStep(c, name)
}

func (c *Config) SubTest(action TestAction) {
	c.Hooks.ApplyBeforeSubTest(c)

	wrapped := action
	for i := len(c.TestWraps) - 1; i >= 0; i-- {
		wrapped = c.TestWraps[i](wrapped)
	}

	wrapped(c)

	c.Hooks.ApplyAfterSubTest(c)
}
