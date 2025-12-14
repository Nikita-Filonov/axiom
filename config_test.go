package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestConfig_Step_HooksOrder(t *testing.T) {
	var calls []string

	cfg := &axiom.Config{
		Hooks: axiom.Hooks{
			BeforeStep: []axiom.StepHook{
				func(_ *axiom.Config, name string) { calls = append(calls, "before:"+name) },
			},
			AfterStep: []axiom.StepHook{
				func(_ *axiom.Config, name string) { calls = append(calls, "after:"+name) },
			},
		},
		SubT: t,
	}

	cfg.Step("X", func() { calls = append(calls, "body") })

	assert.Equal(t,
		[]string{"before:X", "body", "after:X"},
		calls,
	)
}

func TestConfig_Step_WrapsCalled(t *testing.T) {
	var order []string

	rt := axiom.NewRuntime(
		axiom.WithRuntimeStepWrap(
			func(name string, next axiom.StepAction) axiom.StepAction {
				return func() {
					order = append(order, "wrap1")
					next()
				}
			},
		),
		axiom.WithRuntimeStepWrap(
			func(name string, next axiom.StepAction) axiom.StepAction {
				return func() {
					order = append(order, "wrap2")
					next()
				}
			},
		),
	)

	cfg := &axiom.Config{
		Runtime: rt,
		SubT:    t,
	}

	cfg.Step("A", func() {
		order = append(order, "body")
	})

	assert.Equal(t,
		[]string{"wrap1", "wrap2", "body"},
		order,
	)
}

func TestConfig_Test_WrapsCalled(t *testing.T) {
	var order []string

	rt := axiom.NewRuntime(
		axiom.WithRuntimeTestWrap(
			func(next axiom.TestAction) axiom.TestAction {
				return func(c *axiom.Config) {
					order = append(order, "wrap1")
					next(c)
				}
			},
		),
		axiom.WithRuntimeTestWrap(
			func(next axiom.TestAction) axiom.TestAction {
				return func(c *axiom.Config) {
					order = append(order, "wrap2")
					next(c)
				}
			},
		),
	)

	cfg := &axiom.Config{
		Runtime: rt,
		SubT:    t,
	}

	cfg.Test(func(_ *axiom.Config) {
		order = append(order, "body")
	})

	assert.Equal(t,
		[]string{"wrap1", "wrap2", "body"},
		order,
	)
}

func TestConfig_Test_HooksOrder(t *testing.T) {
	var calls []string

	cfg := &axiom.Config{
		Name: "T",
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{
				func(_ *axiom.Config) { calls = append(calls, "before") },
			},
			AfterTest: []axiom.TestHook{
				func(_ *axiom.Config) { calls = append(calls, "after") },
			},
		},
		SubT: t,
	}

	cfg.Test(func(_ *axiom.Config) {
		calls = append(calls, "body")
	})

	assert.Equal(t,
		[]string{"before", "body", "after"},
		calls,
	)
}

func TestConfig_ApplyPlugins_OrderAndRunnerCase(t *testing.T) {
	var calls []string

	r := axiom.NewRunner(
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			calls = append(calls, "runner")
		}),
	)

	c := axiom.NewCase(
		axiom.WithCasePlugins(func(cfg *axiom.Config) {
			calls = append(calls, "case")
		}),
	)

	cfg := &axiom.Config{
		Runner: r,
		Case:   &c,
	}

	cfg.ApplyPlugins()

	assert.Equal(t, []string{"runner", "case"}, calls)
}

func TestConfig_ApplyExecutionPolicy_Parallel_DoesNotPanic(t *testing.T) {
	cfg := &axiom.Config{
		RootT:    t,
		Runtime:  axiom.NewRuntime(),
		Parallel: axiom.Parallel{Enabled: true},
	}

	cfg.ApplyExecutionPolicy()
}
