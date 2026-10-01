package testexplain_test

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testexplain"
	"github.com/stretchr/testify/require"
)

func explainPluginZ(*axiom.Config)    {}
func explainPluginA(*axiom.Config)    {}
func explainBeforeTest(*axiom.Config) {}
func explainEventSink(axiom.Event)    {}

func TestExplainRunner_PreservesPluginOrderAcrossJoin(t *testing.T) {
	base := axiom.NewRunner(axiom.WithRunnerPlugins(explainPluginZ))
	overlay := axiom.NewRunner(axiom.WithRunnerPlugins(explainPluginA))
	explanation := testexplain.ExplainRunner(base.Join(overlay))

	names := explanation.Plugins.Runner.Names
	require.Len(t, names, 2)
	require.True(t, strings.HasSuffix(names[0], ".explainPluginZ"))
	require.True(t, strings.HasSuffix(names[1], ".explainPluginA"))
	require.Equal(t, 1, explanation.Runner.Parent.Plugins.Runner.Count)
	require.Equal(t, 1, explanation.Runner.Overlay.Plugins.Runner.Count)
}

func TestExplainRunner_SharedSourceAndCycle(t *testing.T) {
	root := axiom.NewRunner()
	shared := axiom.NewRunner()
	root.Parent = shared
	root.Overlay = shared

	explanation := testexplain.ExplainRunner(root)
	require.False(t, explanation.Runner.Parent.Runner.Cycle)
	require.False(t, explanation.Runner.Overlay.Runner.Cycle)

	shared.Parent = root
	explanation = testexplain.ExplainRunner(root)
	require.True(t, explanation.Runner.Parent.Runner.Parent.Runner.Cycle)
	require.True(t, explanation.Runner.Overlay.Runner.Parent.Runner.Cycle)
	_, err := json.Marshal(explanation)
	require.NoError(t, err)
}

func TestExplainConfig_EmptySourcesAndTypedNilParams(t *testing.T) {
	empty := testexplain.ExplainConfig(&axiom.Config{})
	require.Equal(t, testexplain.ExplanationKindConfig, empty.Kind)
	require.NotNil(t, empty.Runner)
	require.Nil(t, empty.Case)
	require.Equal(t, 0, empty.Plugins.Total)

	var params *int
	c := axiom.NewCase(axiom.WithCaseParams(params))
	explanation := testexplain.ExplainConfig(&axiom.Config{Case: &c})
	require.Equal(t, "*int", explanation.Case.ParamsType)
}

func TestExplainerSnapshot_ClonesRunnerAndCaseDetails(t *testing.T) {
	fixture := func(*axiom.Config) (any, func(), error) { return nil, nil, nil }
	base := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaLabel("owner", "runner")),
		axiom.WithRunnerContext(axiom.WithContextData("z", 1), axiom.WithContextData("a", 2)),
		axiom.WithRunnerFixture("runner", fixture),
		axiom.WithRunnerPlugins(explainPluginZ),
		axiom.WithRunnerHooks(axiom.WithBeforeTest(explainBeforeTest)),
		axiom.WithRunnerRuntime(axiom.WithRuntimeEventSink(explainEventSink)),
	)
	joined := base.Join(axiom.NewRunner())
	c := axiom.NewCase(
		axiom.WithCaseMeta(axiom.WithMetaLabel("owner", "case")),
		axiom.WithCaseContext(axiom.WithContextData("case", 3)),
		axiom.WithCaseFixture("case", fixture),
		axiom.WithCasePlugins(explainPluginA),
		axiom.WithCaseHooks(axiom.WithBeforeTest(explainBeforeTest)),
		axiom.WithCaseRuntime(axiom.WithRuntimeEventSink(explainEventSink)),
	)
	explanation := testexplain.ExplainConfig(joined.BuildConfig(t, &c))
	require.Equal(t, []string{"case", "runner"}, explanation.Fixtures)
	require.Equal(t, []string{"a", "z"}, explanation.Runner.Context.DataKeys)
	joined.Meta.Labels["owner"] = "changed runner"
	c.Meta.Labels["owner"] = "changed case"
	delete(joined.Fixtures.Registry, "runner")
	require.Equal(t, "runner", explanation.Runner.Meta.Labels["owner"])
	require.Equal(t, "case", explanation.Case.Meta.Labels["owner"])
	require.Equal(t, []string{"runner"}, explanation.Runner.Fixtures)

	explainer := testexplain.NewExplainer()
	explainer.Record(explanation)
	expected, err := json.Marshal(explainer.Snapshot())
	require.NoError(t, err)

	explanation.Meta.Labels["owner"] = "changed"
	explanation.Runner.Parent.Meta.Labels["owner"] = "changed"
	explanation.Case.Meta.Labels["owner"] = "changed"
	explanation.Context.DataKeys[0] = "changed"
	explanation.Runner.Fixtures[0] = "changed"
	explanation.Case.Fixtures[0] = "changed"
	explanation.Hooks.BeforeTest.Names[0] = "changed"
	explanation.Runtime.EventSinks.Names[0] = "changed"
	explanation.Plugins.Runner.Names[0] = "changed"

	snapshot := explainer.Snapshot()
	snapshot[0].Runner.Meta.Labels["owner"] = "changed again"
	snapshot[0].Case.Meta.Labels["owner"] = "changed again"
	snapshot[0].Runner.Parent.Meta.Labels["owner"] = "changed again"
	snapshot[0].Case.Context.DataKeys[0] = "changed again"
	snapshot[0].Case.Plugins.Names[0] = "changed again"
	snapshot[0].Case.Hooks.BeforeTest.Names[0] = "changed again"
	snapshot[0].Case.Runtime.EventSinks.Names[0] = "changed again"
	actual, err := json.Marshal(explainer.Snapshot())
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestExplainer_ConcurrentRecordAndSnapshot(t *testing.T) {
	const count = 32
	explainer := testexplain.NewExplainer()
	var workers sync.WaitGroup
	workers.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer workers.Done()
			explainer.Record(testexplain.Explanation{Kind: testexplain.ExplanationKindConfig})
			_ = explainer.Snapshot()
		}()
	}
	workers.Wait()
	require.Len(t, explainer.Snapshot(), count)
}

func TestPlugin_RecordsRealAttemptAfterCasePlugins(t *testing.T) {
	explainer := testexplain.NewExplainer()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(testexplain.Plugin(explainer)))
	c := axiom.NewCase(
		axiom.WithCaseName("recorded attempt"),
		axiom.WithCasePlugins(func(cfg *axiom.Config) {
			cfg.Meta.Epic = "set by case plugin"
		}),
	)
	runner.RunCase(t, c, func(cfg *axiom.Config) {})

	snapshots := explainer.Snapshot()
	require.Len(t, snapshots, 1)
	require.Equal(t, "recorded attempt", snapshots[0].Case.Name)
	require.Equal(t, "set by case plugin", snapshots[0].Meta.Epic)
}
