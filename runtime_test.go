package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewRuntime_Defaults(t *testing.T) {
	rt := axiom.NewRuntime()

	assert.Empty(t, rt.TestWraps)
	assert.Empty(t, rt.StepWraps)
	assert.Empty(t, rt.SetupWraps)
	assert.Empty(t, rt.TeardownWraps)

	assert.Empty(t, rt.LogSinks)
	assert.Empty(t, rt.AssertSinks)
	assert.Empty(t, rt.ArtefactSinks)
}

func TestWithRuntimeTestWrap(t *testing.T) {
	wrap := func(next axiom.TestAction) axiom.TestAction {
		return next
	}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeTestWrap(wrap),
	)

	assert.Len(t, rt.TestWraps, 1)
}

func TestWithRuntimeStepWrap(t *testing.T) {
	wrap := func(name string, next axiom.StepAction) axiom.StepAction {
		return next
	}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeStepWrap(wrap),
	)

	assert.Len(t, rt.StepWraps, 1)
}

func TestWithRuntimeSetupWrap(t *testing.T) {
	wrap := func(name string, next axiom.SetupAction) axiom.SetupAction {
		return next
	}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeSetupWrap(wrap),
	)

	assert.Len(t, rt.SetupWraps, 1)
}

func TestWithRuntimeTeardownWrap(t *testing.T) {
	wrap := func(name string, next axiom.TeardownAction) axiom.TeardownAction {
		return next
	}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeTeardownWrap(wrap),
	)

	assert.Len(t, rt.TeardownWraps, 1)
}

func TestWithRuntimeLogSink(t *testing.T) {
	sink := func(l axiom.Log) {}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeLogSink(sink),
	)

	assert.Len(t, rt.LogSinks, 1)
}

func TestWithRuntimeAssertSink(t *testing.T) {
	sink := func(a axiom.Assert) {}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeAssertSink(sink),
	)

	assert.Len(t, rt.AssertSinks, 1)
}

func TestWithRuntimeArtefactSink(t *testing.T) {
	sink := func(a axiom.Artefact) {}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeArtefactSink(sink),
	)

	assert.Len(t, rt.ArtefactSinks, 1)
}

func TestRuntime_EmitIgnoresNil(t *testing.T) {
	rt := axiom.NewRuntime()

	rt.EmitTestWrap(nil)
	rt.EmitStepWrap(nil)
	rt.EmitSetupWrap(nil)
	rt.EmitTeardownWrap(nil)

	rt.EmitLogSink(nil)
	rt.EmitAssertSink(nil)
	rt.EmitArtefactSink(nil)

	assert.Empty(t, rt.TestWraps)
	assert.Empty(t, rt.StepWraps)
	assert.Empty(t, rt.SetupWraps)
	assert.Empty(t, rt.TeardownWraps)

	assert.Empty(t, rt.LogSinks)
	assert.Empty(t, rt.AssertSinks)
	assert.Empty(t, rt.ArtefactSinks)
}

func TestRuntime_LogCallsAllSinks(t *testing.T) {
	var calls int

	rt := axiom.NewRuntime(
		axiom.WithRuntimeLogSink(func(l axiom.Log) { calls++ }),
		axiom.WithRuntimeLogSink(func(l axiom.Log) { calls++ }),
	)

	rt.Log(axiom.Log{Text: "hello"})

	assert.Equal(t, 2, calls)
}

func TestRuntime_AssertCallsAllSinks(t *testing.T) {
	var calls int

	rt := axiom.NewRuntime(
		axiom.WithRuntimeAssertSink(func(a axiom.Assert) { calls++ }),
		axiom.WithRuntimeAssertSink(func(a axiom.Assert) { calls++ }),
	)

	rt.Assert(axiom.Assert{Type: axiom.AssertEqual})

	assert.Equal(t, 2, calls)
}

func TestRuntime_ArtefactCallsAllSinks(t *testing.T) {
	var calls int

	rt := axiom.NewRuntime(
		axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) { calls++ }),
		axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) { calls++ }),
	)

	rt.Artefact(axiom.Artefact{Name: "file"})

	assert.Equal(t, 2, calls)
}

func TestRuntime_StepWrapOrder(t *testing.T) {
	var order []string

	w1 := func(name string, next axiom.StepAction) axiom.StepAction {
		return func() {
			order = append(order, "w1-before")
			next()
			order = append(order, "w1-after")
		}
	}

	w2 := func(name string, next axiom.StepAction) axiom.StepAction {
		return func() {
			order = append(order, "w2-before")
			next()
			order = append(order, "w2-after")
		}
	}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeStepWrap(w1),
		axiom.WithRuntimeStepWrap(w2),
	)

	rt.Step("step", func() {
		order = append(order, "step")
	})

	assert.Equal(t, []string{
		"w1-before",
		"w2-before",
		"step",
		"w2-after",
		"w1-after",
	}, order)
}

func TestRuntime_TestWrapOrder(t *testing.T) {
	var order []string

	w1 := func(next axiom.TestAction) axiom.TestAction {
		return func(c *axiom.Config) {
			order = append(order, "w1-before")
			next(c)
			order = append(order, "w1-after")
		}
	}

	w2 := func(next axiom.TestAction) axiom.TestAction {
		return func(c *axiom.Config) {
			order = append(order, "w2-before")
			next(c)
			order = append(order, "w2-after")
		}
	}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeTestWrap(w1),
		axiom.WithRuntimeTestWrap(w2),
	)

	rt.Test(nil, func(_ *axiom.Config) {
		order = append(order, "test")
	})

	assert.Equal(t, []string{
		"w1-before",
		"w2-before",
		"test",
		"w2-after",
		"w1-after",
	}, order)
}

func TestRuntime_SetupWrapOrder(t *testing.T) {
	var order []string

	w1 := func(name string, next axiom.SetupAction) axiom.SetupAction {
		return func() {
			order = append(order, "w1-before")
			next()
			order = append(order, "w1-after")
		}
	}

	w2 := func(name string, next axiom.SetupAction) axiom.SetupAction {
		return func() {
			order = append(order, "w2-before")
			next()
			order = append(order, "w2-after")
		}
	}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeSetupWrap(w1),
		axiom.WithRuntimeSetupWrap(w2),
	)

	rt.Setup("setup", func() {
		order = append(order, "setup")
	})

	assert.Equal(t, []string{
		"w1-before",
		"w2-before",
		"setup",
		"w2-after",
		"w1-after",
	}, order)
}

func TestRuntime_TeardownWrapOrder(t *testing.T) {
	var order []string

	w1 := func(name string, next axiom.TeardownAction) axiom.TeardownAction {
		return func() {
			order = append(order, "w1-before")
			next()
			order = append(order, "w1-after")
		}
	}

	w2 := func(name string, next axiom.TeardownAction) axiom.TeardownAction {
		return func() {
			order = append(order, "w2-before")
			next()
			order = append(order, "w2-after")
		}
	}

	rt := axiom.NewRuntime(
		axiom.WithRuntimeTeardownWrap(w1),
		axiom.WithRuntimeTeardownWrap(w2),
	)

	rt.Teardown("teardown", func() {
		order = append(order, "teardown")
	})

	assert.Equal(t, []string{
		"w1-before",
		"w2-before",
		"teardown",
		"w2-after",
		"w1-after",
	}, order)
}

func TestRuntimeJoin(t *testing.T) {
	rt1 := axiom.NewRuntime(
		axiom.WithRuntimeLogSink(func(l axiom.Log) {}),
		axiom.WithRuntimeAssertSink(func(a axiom.Assert) {}),
		axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction { return next }),
		axiom.WithRuntimeSetupWrap(func(name string, next axiom.SetupAction) axiom.SetupAction { return next }),
	)

	rt2 := axiom.NewRuntime(
		axiom.WithRuntimeAssertSink(func(a axiom.Assert) {}),
		axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) {}),
		axiom.WithRuntimeStepWrap(func(name string, next axiom.StepAction) axiom.StepAction { return next }),
		axiom.WithRuntimeSetupWrap(func(name string, next axiom.SetupAction) axiom.SetupAction { return next }),
		axiom.WithRuntimeTeardownWrap(func(name string, next axiom.TeardownAction) axiom.TeardownAction { return next }),
	)

	joined := rt1.Join(rt2)

	assert.Len(t, joined.StepWraps, 1)
	assert.Len(t, joined.TestWraps, 1)
	assert.Len(t, joined.SetupWraps, 2)
	assert.Len(t, joined.TeardownWraps, 1)

	assert.Len(t, joined.LogSinks, 1)
	assert.Len(t, joined.AssertSinks, 2)
	assert.Len(t, joined.ArtefactSinks, 1)
}

func TestRuntime_Emitters_DoNotMutateRuntime(t *testing.T) {
	rt := axiom.NewRuntime()

	rt.Log(axiom.Log{Text: "log"})
	rt.Assert(axiom.Assert{Type: axiom.AssertTrue})
	rt.Artefact(axiom.Artefact{Name: "file"})

	assert.Empty(t, rt.LogSinks)
	assert.Empty(t, rt.AssertSinks)
	assert.Empty(t, rt.ArtefactSinks)
}
