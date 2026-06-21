package testtracing_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtracing"
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
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if records[0].Case.ID != "id" || records[0].Case.Name != "case" {
		t.Fatalf("unexpected record case: %#v", records[0])
	}
	if records[0].Meta.Epic != "epic" {
		t.Fatalf("unexpected record meta: %#v", records[0].Meta)
	}
	if len(records[0].Events) != 1 || records[0].Events[0].Type != axiom.EventTypeCaseStart {
		t.Fatalf("unexpected record events: %#v", records[0].Events)
	}
}

func TestPlugin_PreservesConfigEventsAsIs(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{}

	testtracing.Plugin(trace)(cfg)

	cfg.Event(axiom.Event{Type: axiom.EventTypeLog, Message: "raw"})

	records := trace.Snapshot()
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if len(records[0].Events) != 1 {
		t.Fatalf("expected one event, got %d", len(records[0].Events))
	}
	if records[0].Events[0] != (axiom.Event{Type: axiom.EventTypeLog, Message: "raw"}) {
		t.Fatalf("unexpected event: %#v", records[0].Events[0])
	}
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
	if len(records) != 2 {
		t.Fatalf("expected two records, got %d", len(records))
	}
	if records[0].Case.Name != "A" || records[1].Case.Name != "B" {
		t.Fatalf("unexpected record order: %#v", records)
	}
	if len(records[0].Events) != 2 || records[0].Events[0].Type != axiom.EventTypeCaseStart || records[0].Events[1].Type != axiom.EventTypeCaseFinish {
		t.Fatalf("unexpected A events: %#v", records[0].Events)
	}
	if len(records[1].Events) != 2 || records[1].Events[0].Type != axiom.EventTypeCaseStart || records[1].Events[1].Type != axiom.EventTypeCaseFinish {
		t.Fatalf("unexpected B events: %#v", records[1].Events)
	}
}

func TestPlugin_DoesNotCollectRunnerRuntimeEvents(t *testing.T) {
	trace := testtracing.NewTrace()
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("resource", func(r *axiom.Runner) (any, func(), error) {
			return "ok", nil, nil
		}),
	)
	c := axiom.NewCase(axiom.WithCaseName("case"))
	cfg := runner.BuildConfig(t, &c)

	testtracing.Plugin(trace)(cfg)

	value := axiom.MustResource[string](runner, "resource")
	if value != "ok" {
		t.Fatalf("unexpected resource value: %s", value)
	}

	records := trace.Snapshot()
	if len(records) != 0 {
		t.Fatalf("expected no records for runner runtime events, got %#v", records)
	}
}

func TestPlugin_DuplicateApplicationsCreateIndependentRecords(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "case"},
	}
	plugin := testtracing.Plugin(trace)

	plugin(cfg)
	plugin(cfg)
	cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))

	records := trace.Snapshot()
	if len(records) != 2 {
		t.Fatalf("expected two records from duplicate plugin application, got %d: %#v", len(records), records)
	}
	for _, record := range records {
		if record.Case.Name != "case" {
			t.Fatalf("unexpected record case: %#v", record.Case)
		}
		if len(record.Events) != 1 || record.Events[0].Type != axiom.EventTypeCaseStart {
			t.Fatalf("unexpected record events: %#v", record.Events)
		}
	}
}

func TestPlugin_ClosesSinkOnTestingCleanup(t *testing.T) {
	trace := testtracing.NewTrace()
	var cfg *axiom.Config

	t.Run("case", func(t *testing.T) {
		cfg = &axiom.Config{
			RootT:   t,
			Runtime: axiom.NewRuntime(),
		}
		testtracing.Plugin(trace)(cfg)

		cfg.Runtime.Test(cfg, func(_ *axiom.Config) {})
		cfg.Event(axiom.NewEvent(axiom.EventTypeLog))
	})

	cfg.Event(axiom.NewEvent(axiom.EventTypeAssert))

	records := trace.Snapshot()
	if len(records) != 1 {
		t.Fatalf("expected one record before cleanup, got %d", len(records))
	}
	if len(records[0].Events) != 1 || records[0].Events[0].Type != axiom.EventTypeLog {
		t.Fatalf("unexpected record events: %#v", records[0].Events)
	}
}

func TestPlugin_KeepsSinkActiveWhenTestingTUnavailable(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{Runtime: axiom.NewRuntime()}
	testtracing.Plugin(trace)(cfg)

	cfg.Runtime.Test(cfg, func(_ *axiom.Config) {})
	cfg.Event(axiom.NewEvent(axiom.EventTypeLog))

	records := trace.Snapshot()
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if len(records[0].Events) != 1 || records[0].Events[0].Type != axiom.EventTypeLog {
		t.Fatalf("unexpected record events: %#v", records[0].Events)
	}
}

func TestTraceSnapshot_IsIndependent(t *testing.T) {
	trace := testtracing.NewTrace()
	cfg := &axiom.Config{}
	testtracing.Plugin(trace)(cfg)
	cfg.Event(axiom.NewEvent(axiom.EventTypeLog))

	snapshot := trace.Snapshot()
	snapshot[0].Events[0].Type = axiom.EventTypeAssert

	again := trace.Snapshot()
	if again[0].Events[0].Type != axiom.EventTypeLog {
		t.Fatalf("snapshot mutation changed trace: %s", again[0].Events[0].Type)
	}
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
	if again[0].Case.Name != "case" {
		t.Fatalf("snapshot mutation changed case name: %s", again[0].Case.Name)
	}
	if again[0].Case.Meta.Labels["case"] != "value" {
		t.Fatalf("snapshot mutation changed case meta: %#v", again[0].Case.Meta)
	}
	if again[0].Meta.Epic != "epic" || again[0].Meta.Labels["k"] != "v" {
		t.Fatalf("snapshot mutation changed meta: %#v", again[0].Meta)
	}
	if again[0].Events[0].Type != axiom.EventTypeLog {
		t.Fatalf("snapshot mutation changed events: %#v", again[0].Events)
	}
}
