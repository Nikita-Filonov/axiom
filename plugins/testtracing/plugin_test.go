package testtracing_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_CollectsConfigEvents(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{
		Case: &axiom.Case{ID: "id", Name: "case"},
		Meta: axiom.NewMeta(axiom.WithMetaEpic("epic")),
	}

	testtracing.Plugin(trace)(cfg)

	cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))

	records := trace.Snapshot()
	require.Len(t, records, 1)
	require.Equal(t, "id", records[0].Case.ID)
	require.Equal(t, "case", records[0].Case.Name)
	require.Equal(t, "epic", records[0].Meta.Epic)
	require.Len(t, records[0].Events, 1)
	require.Equal(t, axiom.EventTypeCaseStart, records[0].Events[0].Type)
}

func TestPlugin_PreservesConfigEventsAsIs(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{}

	testtracing.Plugin(trace)(cfg)

	cfg.Event(axiom.Event{Type: axiom.EventTypeLog, Message: "raw"})

	records := trace.Snapshot()
	require.Len(t, records, 1)
	require.Len(t, records[0].Events, 1)
	require.Equal(t, axiom.Event{Type: axiom.EventTypeLog, Message: "raw"}, records[0].Events[0])
}

func TestPlugin_GroupsEventsByConfig(t *testing.T) {
	trace := testtracing.NewTrace()
	cfgA := &axiom.Config{
		Case: &axiom.Case{Name: "A"},
	}
	cfgB := &axiom.Config{
		Case: &axiom.Case{Name: "B"},
	}

	plugin := testtracing.Plugin(trace)
	plugin(cfgA)
	plugin(cfgB)

	cfgA.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
	cfgB.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
	cfgA.Event(axiom.NewEvent(axiom.EventTypeCaseFinish))
	cfgB.Event(axiom.NewEvent(axiom.EventTypeCaseFinish))

	records := trace.Snapshot()
	require.Len(t, records, 2)
	require.Equal(t, "A", records[0].Case.Name)
	require.Equal(t, "B", records[1].Case.Name)
	require.Len(t, records[0].Events, 2)
	require.Equal(t, axiom.EventTypeCaseStart, records[0].Events[0].Type)
	require.Equal(t, axiom.EventTypeCaseFinish, records[0].Events[1].Type)
	require.Len(t, records[1].Events, 2)
	require.Equal(t, axiom.EventTypeCaseStart, records[1].Events[0].Type)
	require.Equal(t, axiom.EventTypeCaseFinish, records[1].Events[1].Type)
}

func TestPlugin_DoesNotCollectRunnerRuntimeEvents(t *testing.T) {
	trace := testtracing.NewTrace()
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("resource", func(r *axiom.Runner) (any, func(), error) {
			return "ok", nil, nil
		}),
	)
	cfg := &axiom.Config{Runner: runner, Case: &axiom.Case{Name: "case"}}

	testtracing.Plugin(trace)(cfg)

	value := axiom.MustResource[string](runner, "resource")
	require.Equal(t, "ok", value)

	records := trace.Snapshot()
	require.Empty(t, records)
}

func TestPlugin_DuplicateApplicationsRecordEventsOnce(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "case"},
	}
	plugin := testtracing.Plugin(trace)

	plugin(cfg)
	plugin(cfg)
	event := axiom.Event{Type: axiom.EventTypeLog, Message: "repeated event"}
	cfg.Event(event)
	// A separately constructed plugin using the same Trace is also a duplicate.
	testtracing.Plugin(trace)(cfg)
	cfg.Event(event)

	records := trace.Snapshot()
	require.Len(t, records, 1)
	assert.Equal(t, "case", records[0].Case.Name)
	assert.Equal(t, []axiom.Event{event, event}, records[0].Events)
	assert.Len(t, cfg.Runtime.EventSinks, 1)
	assert.Len(t, cfg.Runtime.TestWraps, 1)
}

func TestPlugin_ClosesSinkOnTestingCleanup(t *testing.T) {
	trace := testtracing.NewTrace()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(testtracing.Plugin(trace)))
	var cfg *axiom.Config

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("case")), func(current *axiom.Config) {
		cfg = current
		cfg.Event(axiom.NewEvent(axiom.EventTypeLog))
	})

	cfg.Event(axiom.NewEvent(axiom.EventTypeAssert))

	records := trace.Snapshot()
	require.Len(t, records, 1)
	var types []axiom.EventType
	for _, event := range records[0].Events {
		types = append(types, event.Type)
	}
	require.Equal(t, []axiom.EventType{axiom.EventTypeCaseStart, axiom.EventTypeLog, axiom.EventTypeCaseFinish}, types)
}

func TestPlugin_KeepsSinkActiveWhenTestingTUnavailable(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{Runtime: axiom.NewRuntime()}
	testtracing.Plugin(trace)(cfg)

	require.Len(t, cfg.Runtime.TestWraps, 1)
	cfg.Runtime.TestWraps[0](func(*axiom.Config) {})(cfg)
	cfg.Event(axiom.NewEvent(axiom.EventTypeLog))

	records := trace.Snapshot()
	require.Len(t, records, 1)
	require.Len(t, records[0].Events, 1)
	require.Equal(t, axiom.EventTypeLog, records[0].Events[0].Type)
}

func TestTraceSnapshot_IsIndependent(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{}
	testtracing.Plugin(trace)(cfg)
	cfg.Event(axiom.NewEvent(axiom.EventTypeLog))

	snapshot := trace.Snapshot()
	snapshot[0].Events[0].Type = axiom.EventTypeAssert

	again := trace.Snapshot()
	require.Equal(t, axiom.EventTypeLog, again[0].Events[0].Type)
}

func TestTraceSnapshot_CopiesRecords(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{
		Case: &axiom.Case{
			Name: "case",
			Meta: axiom.NewMeta(
				axiom.WithMetaLabel("case", "value"),
			),
		},
		Meta: axiom.NewMeta(
			axiom.WithMetaEpic("epic"),
			axiom.WithMetaLabel("k", "v"),
		),
	}
	testtracing.Plugin(trace)(cfg)
	cfg.Event(axiom.NewEvent(axiom.EventTypeLog))

	snapshot := trace.Snapshot()
	snapshot[0].Case.Name = "changed"
	snapshot[0].Case.Meta.Labels["case"] = "changed"
	snapshot[0].Meta.Epic = "changed"
	snapshot[0].Meta.Labels["k"] = "changed"
	snapshot[0].Events[0].Type = axiom.EventTypeAssert

	again := trace.Snapshot()
	require.Equal(t, "case", again[0].Case.Name)
	require.Equal(t, "value", again[0].Case.Meta.Labels["case"])
	require.Equal(t, "epic", again[0].Meta.Epic)
	require.Equal(t, "v", again[0].Meta.Labels["k"])
	require.Equal(t, axiom.EventTypeLog, again[0].Events[0].Type)
}
