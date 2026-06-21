package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_T(t *testing.T) {
	subT := &testing.T{}

	cfg := &axiom.Config{
		RootT: t,
		SubT:  subT,
	}

	assert.Same(t, subT, cfg.T())

	cfg.SubT = nil
	assert.Same(t, t, cfg.T())

	cfg.RootT = nil
	assert.Nil(t, cfg.T())
}

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
		Case: &axiom.Case{Name: "T"},
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

func TestConfig_Log_DelegatesToRuntimeSink(t *testing.T) {
	var received axiom.Log
	var events []axiom.Event

	rt := axiom.NewRuntime(
		axiom.WithRuntimeLogSink(func(l axiom.Log) { received = l }),
		axiom.WithRuntimeEventSink(func(e axiom.Event) { events = append(events, e) }),
	)
	cfg := &axiom.Config{Runtime: rt}
	log := axiom.Log{Text: "hello"}

	cfg.Log(log)

	assert.Equal(t, log, received)
	require.Len(t, events, 1)
	assert.Equal(t, axiom.EventTypeLog, events[0].Type)
	assert.Equal(t, "hello", events[0].Message)
}

func TestConfig_Assert_DelegatesToRuntimeSink(t *testing.T) {
	var received axiom.Assert
	var events []axiom.Event

	rt := axiom.NewRuntime(
		axiom.WithRuntimeAssertSink(func(a axiom.Assert) { received = a }),
		axiom.WithRuntimeEventSink(func(e axiom.Event) { events = append(events, e) }),
	)
	cfg := &axiom.Config{Runtime: rt}
	input := axiom.Assert{
		Type:    axiom.AssertEqual,
		Message: "test",
	}

	cfg.Assert(input)

	assert.Equal(t, input, received)
	require.Len(t, events, 1)
	assert.Equal(t, axiom.EventTypeAssert, events[0].Type)
	assert.Equal(t, "test", events[0].Message)
}

func TestConfig_Artefact_DelegatesToRuntimeSink(t *testing.T) {
	var received axiom.Artefact
	var events []axiom.Event

	rt := axiom.NewRuntime(
		axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) { received = a }),
		axiom.WithRuntimeEventSink(func(e axiom.Event) { events = append(events, e) }),
	)
	cfg := &axiom.Config{Runtime: rt}
	art := axiom.NewTextArtefact("file", "payload")

	cfg.Artefact(art)

	assert.Equal(t, art, received)
	require.Len(t, events, 1)
	assert.Equal(t, axiom.EventTypeArtefact, events[0].Type)
	assert.Equal(t, axiom.ArtefactTypeText.String(), events[0].Name)
	assert.Equal(t, "file", events[0].Message)
}

func TestConfig_Setup_WrapsCalled(t *testing.T) {
	var order []string

	rt := axiom.NewRuntime(
		axiom.WithRuntimeSetupWrap(
			func(name string, next axiom.SetupAction) axiom.SetupAction {
				return func() {
					order = append(order, "wrap1")
					next()
				}
			},
		),
		axiom.WithRuntimeSetupWrap(
			func(name string, next axiom.SetupAction) axiom.SetupAction {
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

	cfg.Setup("S", func() {
		order = append(order, "body")
	})

	assert.Equal(t,
		[]string{"wrap1", "wrap2", "body"},
		order,
	)
}

func TestConfig_Teardown_WrapsCalled(t *testing.T) {
	var order []string

	rt := axiom.NewRuntime(
		axiom.WithRuntimeTeardownWrap(
			func(name string, next axiom.TeardownAction) axiom.TeardownAction {
				return func() {
					order = append(order, "wrap1")
					next()
				}
			},
		),
		axiom.WithRuntimeTeardownWrap(
			func(name string, next axiom.TeardownAction) axiom.TeardownAction {
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

	cfg.Teardown("T", func() {
		order = append(order, "body")
	})

	assert.Equal(t,
		[]string{"wrap1", "wrap2", "body"},
		order,
	)
}

func TestConfig_Setup_DoesNotCallStepHooks(t *testing.T) {
	var called bool

	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(),
		Hooks: axiom.Hooks{
			BeforeStep: []axiom.StepHook{
				func(_ *axiom.Config, _ string) { called = true },
			},
			AfterStep: []axiom.StepHook{
				func(_ *axiom.Config, _ string) { called = true },
			},
		},
		SubT: t,
	}

	cfg.Setup("setup", func() {})

	assert.False(t, called)
}

func TestConfig_Event_DelegatesAsIs(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
	}

	cfg.Event(axiom.Event{Type: axiom.EventTypeLog, Message: "raw"})

	require.Len(t, events, 1)
	assert.Equal(t, axiom.Event{Type: axiom.EventTypeLog, Message: "raw"}, events[0])
}

func TestConfig_Test_EmitsStartAndFinishFacts(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: t,
	}

	cfg.Test(func(_ *axiom.Config) {})

	require.Len(t, events, 2)
	assert.Equal(t, axiom.EventTypeCaseStart, events[0].Type)
	assert.Equal(t, axiom.EventTypeCaseFinish, events[1].Type)
}

func TestConfig_TestPanic_EmitsPanicFact(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "case"},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: &testing.T{},
	}

	cfg.Test(func(_ *axiom.Config) { panic("boom") })

	require.Len(t, events, 3)
	assert.Equal(t, axiom.EventTypeCaseStart, events[0].Type)
	assert.Equal(t, axiom.EventTypeCasePanic, events[1].Type)
	assert.Equal(t, "boom", events[1].Message)
	assert.Equal(t, axiom.EventTypeCaseFinish, events[2].Type)
}

func TestConfig_StepFinish_DoesNotInferStatusFromTestingT(t *testing.T) {
	fakeT := &testing.T{}
	fakeT.Fail()

	var events []axiom.Event
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: fakeT,
	}

	cfg.Step("clean step", func() {})

	require.Len(t, events, 2)
	assert.Equal(t, axiom.EventTypeStepStart, events[0].Type)
	assert.Equal(t, axiom.EventTypeStepFinish, events[1].Type)
}

func TestConfig_StepPanic_EmitsPanicFact(t *testing.T) {
	fakeT := &testing.T{}

	var events []axiom.Event
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: fakeT,
	}

	cfg.Step("panic step", func() { panic("boom") })

	require.Len(t, events, 3)
	assert.Equal(t, axiom.EventTypeStepStart, events[0].Type)
	assert.Equal(t, axiom.EventTypeStepPanic, events[1].Type)
	assert.Equal(t, "boom", events[1].Message)
	assert.Equal(t, axiom.EventTypeStepFinish, events[2].Type)
}

func TestConfig_Setup_EmitsStartAndFinishFacts(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: t,
	}

	cfg.Setup("setup", func() {})

	require.Len(t, events, 2)
	assert.Equal(t, axiom.EventTypeSetupStart, events[0].Type)
	assert.Equal(t, "setup", events[0].Name)
	assert.Equal(t, axiom.EventTypeSetupFinish, events[1].Type)
	assert.Equal(t, "setup", events[1].Name)
}

func TestConfig_SetupPanic_EmitsPanicFact(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: &testing.T{},
	}

	cfg.Setup("setup", func() { panic("boom") })

	require.Len(t, events, 3)
	assert.Equal(t, axiom.EventTypeSetupStart, events[0].Type)
	assert.Equal(t, axiom.EventTypeSetupPanic, events[1].Type)
	assert.Equal(t, "boom", events[1].Message)
	assert.Equal(t, axiom.EventTypeSetupFinish, events[2].Type)
}

func TestConfig_Teardown_EmitsStartAndFinishFacts(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: t,
	}

	cfg.Teardown("teardown", func() {})

	require.Len(t, events, 2)
	assert.Equal(t, axiom.EventTypeTeardownStart, events[0].Type)
	assert.Equal(t, "teardown", events[0].Name)
	assert.Equal(t, axiom.EventTypeTeardownFinish, events[1].Type)
	assert.Equal(t, "teardown", events[1].Name)
}

func TestConfig_TeardownPanic_EmitsPanicFact(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: &testing.T{},
	}

	cfg.Teardown("teardown", func() { panic("boom") })

	require.Len(t, events, 3)
	assert.Equal(t, axiom.EventTypeTeardownStart, events[0].Type)
	assert.Equal(t, axiom.EventTypeTeardownPanic, events[1].Type)
	assert.Equal(t, "boom", events[1].Message)
	assert.Equal(t, axiom.EventTypeTeardownFinish, events[2].Type)
}
