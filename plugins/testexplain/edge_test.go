package testexplain_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testexplain"
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
	if len(names) != 2 || !strings.HasSuffix(names[0], ".explainPluginZ") || !strings.HasSuffix(names[1], ".explainPluginA") {
		t.Fatalf("plugins lost registration order: %v", names)
	}
	if explanation.Runner.Parent.Plugins.Runner.Count != 1 || explanation.Runner.Overlay.Plugins.Runner.Count != 1 {
		t.Fatal("plugin sources were not preserved")
	}
}

func TestExplainRunner_SharedSourceAndCycle(t *testing.T) {
	root := axiom.NewRunner()
	shared := axiom.NewRunner()
	root.Parent = shared
	root.Overlay = shared

	explanation := testexplain.ExplainRunner(root)
	if explanation.Runner.Parent.Runner.Cycle || explanation.Runner.Overlay.Runner.Cycle {
		t.Fatal("a shared source was marked as a cycle")
	}

	shared.Parent = root
	explanation = testexplain.ExplainRunner(root)
	if !explanation.Runner.Parent.Runner.Parent.Runner.Cycle || !explanation.Runner.Overlay.Runner.Parent.Runner.Cycle {
		t.Fatal("cycle marker missing from a Runner source path")
	}
	if _, err := json.Marshal(explanation); err != nil {
		t.Fatalf("cyclic source explanation is not serializable: %v", err)
	}
}

func TestExplainConfig_EmptySourcesAndTypedNilParams(t *testing.T) {
	empty := testexplain.ExplainConfig(&axiom.Config{})
	if empty.Kind != testexplain.ExplanationKindConfig || empty.Runner == nil || empty.Case != nil || empty.Plugins.Total != 0 {
		t.Fatalf("unexpected empty config explanation: %#v", empty)
	}

	var params *int
	c := axiom.NewCase(axiom.WithCaseParams(params))
	explanation := testexplain.ExplainConfig(&axiom.Config{Case: &c})
	if explanation.Case.ParamsType != "*int" {
		t.Fatalf("typed nil parameter lost its type: %q", explanation.Case.ParamsType)
	}
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
	if strings.Join(explanation.Fixtures, ",") != "case,runner" {
		t.Fatalf("unexpected fixture order: %v", explanation.Fixtures)
	}
	if strings.Join(explanation.Runner.Context.DataKeys, ",") != "a,z" {
		t.Fatalf("context keys are not sorted: %v", explanation.Runner.Context.DataKeys)
	}
	joined.Meta.Labels["owner"] = "changed runner"
	c.Meta.Labels["owner"] = "changed case"
	delete(joined.Fixtures.Registry, "runner")
	if explanation.Runner.Meta.Labels["owner"] != "runner" || explanation.Case.Meta.Labels["owner"] != "case" || strings.Join(explanation.Runner.Fixtures, ",") != "runner" {
		t.Fatal("explanation changed after its Runner or Case was modified")
	}

	explainer := testexplain.NewExplainer()
	explainer.Record(explanation)
	expected, err := json.Marshal(explainer.Snapshot())
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("mutating an explanation or snapshot changed the recorded data")
	}
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
	if got := len(explainer.Snapshot()); got != count {
		t.Fatalf("expected %d concurrent records, got %d", count, got)
	}
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
	if len(snapshots) != 1 || snapshots[0].Case.Name != "recorded attempt" || snapshots[0].Meta.Epic != "set by case plugin" {
		t.Fatalf("unexpected recorded attempt: %#v", snapshots)
	}
}
