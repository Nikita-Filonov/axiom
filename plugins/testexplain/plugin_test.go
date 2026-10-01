package testexplain_test

import (
	"encoding/json"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testexplain"
	"github.com/stretchr/testify/require"
)

func TestExplainConfig_IncludesRuntimeEventSinks(t *testing.T) {
	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {}),
		),
	}

	explanation := testexplain.ExplainConfig(cfg)

	require.Equal(t, 1, explanation.Runtime.EventSinks.Count)
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

	require.Equal(t, testexplain.ExplanationKindRunner, explanation.Kind)
	require.Len(t, explanation.Runner.Fixtures, 1)
	require.Equal(t, "fixture", explanation.Runner.Fixtures[0])
	require.Len(t, explanation.Runner.Resources, 1)
	require.Equal(t, "resource", explanation.Runner.Resources[0])
	require.Equal(t, 1, explanation.Plugins.Total)
	require.Equal(t, 1, explanation.Hooks.BeforeAll.Count)
	require.Equal(t, 1, explanation.Hooks.AfterAll.Count)
	require.Equal(t, 1, explanation.Runtime.EventSinks.Count)
	require.Equal(t, 2, explanation.Retry.Times)
	require.True(t, explanation.Parallel.Enabled)
	require.Len(t, explanation.Context.DataKeys, 1)
	require.Equal(t, "key", explanation.Context.DataKeys[0])
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

	require.NotNil(t, explanation.Runner.Parent)
	require.NotNil(t, explanation.Runner.Overlay)
	require.NotNil(t, explanation.Runner.Overlay.Runner.Parent)
	require.NotNil(t, explanation.Runner.Overlay.Runner.Overlay)
	require.Equal(t, "base", explanation.Runner.Parent.Meta.Epic)
	require.Equal(t, 2, explanation.Runner.Parent.Retry.Times)
	require.Equal(t, 3, explanation.Runner.Overlay.Retry.Times)
	require.Equal(t, 3, explanation.Runner.Retry.Times)
	require.Equal(t, 4, explanation.Case.Retry.Times)
	require.Equal(t, 4, explanation.Retry.Times)
	require.Equal(t, "case", explanation.Case.Meta.Story)
	require.Equal(t, "case", explanation.Meta.Story)
	require.Equal(t, "base", explanation.Runner.Meta.Epic)
	_, err := json.Marshal(explanation)
	require.NoError(t, err)
}

func TestExplainRunner_PanicsOnNilRunner(t *testing.T) {
	require.PanicsWithValue(t, "explain: nil *axiom.Runner", func() {
		testexplain.ExplainRunner(nil)
	})
}

func TestExplainConfig_PanicsOnNilConfig(t *testing.T) {
	require.PanicsWithValue(t, "explain: nil *axiom.Config", func() {
		testexplain.ExplainConfig(nil)
	})
}

func TestExplainRunner_MarksCyclicSource(t *testing.T) {
	runner := axiom.NewRunner()
	runner.Parent = runner

	explanation := testexplain.ExplainRunner(runner)
	require.NotNil(t, explanation.Runner.Parent)
	require.True(t, explanation.Runner.Parent.Runner.Cycle)
}

func TestExplainConfig_ReportsParamTypeAndNilPlugin(t *testing.T) {
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(nil))
	c := axiom.NewCase(axiom.WithCaseParams(42))
	explanation := testexplain.ExplainConfig(&axiom.Config{Runner: runner, Case: &c})

	require.Equal(t, "int", explanation.Case.ParamsType)
	require.Equal(t, 1, explanation.Plugins.Runner.Count)
	require.Empty(t, explanation.Plugins.Runner.Names)
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

	require.True(t, called)

	snapshot := explainer.Snapshot()
	require.Len(t, snapshot, 1)
	require.Equal(t, testexplain.ExplanationKindConfig, snapshot[0].Kind)
	require.Equal(t, "before", snapshot[0].Meta.Epic)
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
	require.Equal(t, testexplain.ExplanationKindRunner, again[0].Kind)
	require.Equal(t, "base", again[0].Meta.Epic)
	require.Equal(t, "base", again[0].Runner.Meta.Epic)
	require.Equal(t, "base", again[0].Runner.Parent.Meta.Epic)
}
