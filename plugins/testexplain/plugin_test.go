package testexplain_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testexplain"
)

func TestExplainConfig_IncludesRuntimeEventSinks(t *testing.T) {
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {}),
		),
	}

	explanation := testexplain.ExplainConfig(cfg)

	if explanation.Runtime.EventSinks.Count != 1 {
		t.Fatalf("expected one event sink, got %d", explanation.Runtime.EventSinks.Count)
	}
}

func TestExplainRunner_IncludesRunnerShape(t *testing.T) {
	runnerPlugin := func(cfg *axiom.Config) {}
	r := axiom.NewRunner(
		axiom.WithRunnerFixture("fixture", func(cfg *axiom.Config) (any, func(), error) {
			return "fixture", nil, nil
		}),
		axiom.WithRunnerResource("resource", func(r *axiom.Runner) (any, func(), error) {
			return "resource", nil, nil
		}),
		axiom.WithRunnerPlugins(runnerPlugin),
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) {}),
			axiom.WithAfterAll(func(r *axiom.Runner) {}),
		),
		axiom.WithRunnerRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {}),
		),
		axiom.WithRunnerContext(axiom.WithContextData("key", "value")),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
	)

	explanation := testexplain.ExplainRunner(r)

	if explanation.Kind != testexplain.ExplanationKindRunner {
		t.Fatalf("unexpected explanation kind: %s", explanation.Kind)
	}
	if len(explanation.Runner.Fixtures) != 1 || explanation.Runner.Fixtures[0] != "fixture" {
		t.Fatalf("unexpected runner fixtures: %#v", explanation.Runner.Fixtures)
	}
	if len(explanation.Runner.Resources) != 1 || explanation.Runner.Resources[0] != "resource" {
		t.Fatalf("unexpected runner resources: %#v", explanation.Runner.Resources)
	}
	if explanation.Plugins.Total != 1 {
		t.Fatalf("expected one plugin, got %d", explanation.Plugins.Total)
	}
	if explanation.Hooks.BeforeAll.Count != 1 || explanation.Hooks.AfterAll.Count != 1 {
		t.Fatalf("unexpected hook explanation: %#v", explanation.Hooks)
	}
	if explanation.Runtime.EventSinks.Count != 1 {
		t.Fatalf("expected one event sink, got %d", explanation.Runtime.EventSinks.Count)
	}
	if explanation.Retry.Times != 2 {
		t.Fatalf("unexpected retry times: %d", explanation.Retry.Times)
	}
	if !explanation.Parallel.Enabled {
		t.Fatal("expected parallel to be enabled")
	}
	if len(explanation.Context.DataKeys) != 1 || explanation.Context.DataKeys[0] != "key" {
		t.Fatalf("unexpected context data keys: %#v", explanation.Context.DataKeys)
	}
}

func TestExplainRunner_PanicsOnNilRunner(t *testing.T) {
	defer func() {
		if v := recover(); v != "explain: nil *axiom.Runner" {
			t.Fatalf("unexpected panic: %#v", v)
		}
	}()

	testexplain.ExplainRunner(nil)
}

func TestPlugin_RecordsExplanationBeforeTest(t *testing.T) {
	explainer := testexplain.NewExplainer()
	cfg := &axiom.Config{
		Case:    &axiom.Case{Name: "case"},
		Runner:  axiom.NewRunner(),
		Runtime: axiom.NewRuntime(),
	}

	testexplain.Plugin(explainer)(cfg)

	called := false
	cfg.Runtime.Test(cfg, func(_ *axiom.Config) {
		called = true
	})

	if !called {
		t.Fatal("expected wrapped test action to be called")
	}

	snapshot := explainer.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected one explanation, got %d", len(snapshot))
	}
	if snapshot[0].Kind != testexplain.ExplanationKindConfig {
		t.Fatalf("unexpected explanation kind: %s", snapshot[0].Kind)
	}
}

func TestExplainerSnapshot_IsIndependent(t *testing.T) {
	explainer := testexplain.NewExplainer()
	explainer.Record(testexplain.Explanation{Kind: testexplain.ExplanationKindConfig})

	snapshot := explainer.Snapshot()
	snapshot[0].Kind = testexplain.ExplanationKindRunner

	again := explainer.Snapshot()
	if again[0].Kind != testexplain.ExplanationKindConfig {
		t.Fatalf("snapshot mutation changed explainer: %s", again[0].Kind)
	}
}
