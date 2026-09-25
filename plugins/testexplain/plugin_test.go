package testexplain_test

import (
	"encoding/json"
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

func TestExplainConfig_IncludesJoinHistory(t *testing.T) {
	base := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaEpic("base")),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
	)
	overlayBase := axiom.NewRunner()
	overlay := overlayBase.Join(axiom.NewRunner(axiom.WithRunnerRetry(axiom.WithRetryTimes(3))))
	joined := base.Join(overlay)
	base.Retry.Times = 9
	overlay.Retry.Times = 9

	c := axiom.NewCase(
		axiom.WithCaseMeta(axiom.WithMetaStory("case")),
		axiom.WithCaseRetry(axiom.WithRetryTimes(4)),
	)
	cfg := joined.BuildConfig(t, &c)
	explanation := testexplain.ExplainConfig(cfg)

	if explanation.Runner.Parent == nil || explanation.Runner.Overlay == nil {
		t.Fatal("runner Join inputs missing from explanation")
	}
	if explanation.Runner.Overlay.Runner.Parent == nil || explanation.Runner.Overlay.Runner.Overlay == nil {
		t.Fatal("nested overlay Join missing from explanation")
	}
	if explanation.Runner.Parent.Meta.Epic != "base" {
		t.Fatal("runner parent missing from explanation")
	}
	if explanation.Runner.Parent.Retry.Times != 2 || explanation.Runner.Overlay.Retry.Times != 3 {
		t.Fatal("runner explanation changed after its Join inputs were modified")
	}
	if explanation.Runner.Retry.Times != 3 || explanation.Case.Retry.Times != 4 || explanation.Retry.Times != 4 {
		t.Fatalf("runner, case, or effective retry missing from explanation: %#v", explanation)
	}
	if explanation.Case.Meta.Story != "case" || explanation.Meta.Story != "case" || explanation.Runner.Meta.Epic != "base" {
		t.Fatal("runner, case, or effective metadata missing from explanation")
	}
	if _, err := json.Marshal(explanation); err != nil {
		t.Fatalf("explanation is not JSON serializable: %v", err)
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

func TestExplainConfig_PanicsOnNilConfig(t *testing.T) {
	defer func() {
		if v := recover(); v != "explain: nil *axiom.Config" {
			t.Fatalf("unexpected panic: %#v", v)
		}
	}()

	testexplain.ExplainConfig(nil)
}

func TestExplainRunner_MarksCyclicSource(t *testing.T) {
	runner := axiom.NewRunner()
	runner.Parent = runner

	explanation := testexplain.ExplainRunner(runner)
	if explanation.Runner.Parent == nil || !explanation.Runner.Parent.Runner.Cycle {
		t.Fatal("expected cyclic runner ancestry to be marked")
	}
}

func TestExplainConfig_ReportsParamTypeAndNilPlugin(t *testing.T) {
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(nil))
	c := axiom.NewCase(axiom.WithCaseParams(42))
	explanation := testexplain.ExplainConfig(&axiom.Config{Runner: runner, Case: &c})

	if explanation.Case.ParamsType != "int" {
		t.Fatalf("unexpected parameter type: %q", explanation.Case.ParamsType)
	}
	if explanation.Plugins.Runner.Count != 1 || len(explanation.Plugins.Runner.Names) != 0 {
		t.Fatalf("unexpected nil-plugin summary: %#v", explanation.Plugins.Runner)
	}
}

func TestPlugin_RecordsExplanationBeforeTest(t *testing.T) {
	explainer := testexplain.NewExplainer()
	cfg := &axiom.Config{
		Case:    &axiom.Case{Name: "case"},
		Runner:  axiom.NewRunner(),
		Meta:    axiom.NewMeta(axiom.WithMetaEpic("before")),
		Runtime: axiom.NewRuntime(),
	}

	testexplain.Plugin(explainer)(cfg)

	called := false
	cfg.Runtime.Test(cfg, func(current *axiom.Config) {
		called = true
		current.Meta.Epic = "after"
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
	if snapshot[0].Meta.Epic != "before" {
		t.Fatalf("expected pre-action metadata, got %q", snapshot[0].Meta.Epic)
	}
}

func TestExplainerSnapshot_IsIndependent(t *testing.T) {
	explainer := testexplain.NewExplainer()
	runner := axiom.NewRunner(axiom.WithRunnerMeta(axiom.WithMetaEpic("base")))
	joined := runner.Join(axiom.NewRunner(axiom.WithRunnerMeta(axiom.WithMetaFeature("users"))))
	explanation := testexplain.ExplainRunner(joined)
	explainer.Record(explanation)
	explanation.Meta.Epic = "changed before snapshot"

	snapshot := explainer.Snapshot()
	snapshot[0].Kind = testexplain.ExplanationKindConfig
	snapshot[0].Meta.Epic = "changed"
	snapshot[0].Runner.Meta.Epic = "changed"
	snapshot[0].Runner.Parent.Meta.Epic = "changed"

	again := explainer.Snapshot()
	if again[0].Kind != testexplain.ExplanationKindRunner {
		t.Fatalf("snapshot mutation changed explainer: %s", again[0].Kind)
	}
	if again[0].Meta.Epic != "base" || again[0].Runner.Meta.Epic != "base" || again[0].Runner.Parent.Meta.Epic != "base" {
		t.Fatal("snapshot mutation changed stored Join history")
	}
}
