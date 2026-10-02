package testtracing_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_IndependentTraces(t *testing.T) {
	var first, second testtracing.Trace
	cfg := &axiom.Config{}
	testtracing.Plugin(&first)(cfg)
	testtracing.Plugin(&second)(cfg)
	testtracing.Plugin(&first)(cfg)
	testtracing.Plugin(&second)(cfg)
	event := axiom.Event{Type: axiom.EventTypeLog, Message: "shared"}
	cfg.Event(event)
	assert.Len(t, cfg.Runtime.EventSinks, 2)
	assert.Len(t, cfg.Runtime.TestWraps, 2)
	for _, trace := range []*testtracing.Trace{&first, &second} {
		records := trace.Snapshot()
		require.Len(t, records, 1)
		assert.Equal(t, []axiom.Event{event}, records[0].Events)
	}
	first.AppendToRecord(0, axiom.Event{Type: axiom.EventTypeAssert})
	assert.Len(t, first.Snapshot()[0].Events, 2)
	assert.Len(t, second.Snapshot()[0].Events, 1)
}

func TestPlugin_NilInputs(t *testing.T) {
	assert.PanicsWithValue(t, "testtracing: nil trace", func() { testtracing.Plugin(nil) })
	assert.PanicsWithValue(t, "testtracing: nil config", func() {
		testtracing.Plugin(testtracing.NewTrace())(nil)
	})
}

func TestPlugin_ReinstallationAfterCleanupKeepsSinkClosed(t *testing.T) {
	trace := testtracing.NewTrace()
	plug := testtracing.Plugin(trace)
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(plug))
	var cfg *axiom.Config
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("case")), func(current *axiom.Config) {
		cfg = current
	})
	before := trace.Snapshot()
	require.Len(t, before, 1)
	plug(cfg)
	cfg.Event(axiom.Event{Type: axiom.EventTypeLog, Message: "too late"})
	assert.Equal(t, before, trace.Snapshot())
	assert.Len(t, cfg.Runtime.EventSinks, 1)
	assert.Len(t, cfg.Runtime.TestWraps, 1)
}

func TestPlugin_ConcurrentConfigsAndSnapshots(t *testing.T) {
	trace := testtracing.NewTrace()
	plug := testtracing.Plugin(trace)
	event := axiom.Event{Type: axiom.EventTypeLog, Message: "repeated"}
	var workers sync.WaitGroup
	for i := range 16 {
		workers.Go(func() {
			cfg := &axiom.Config{Case: &axiom.Case{Name: fmt.Sprintf("case-%d", i)}}
			plug(cfg)
			plug(cfg)
			assert.Len(t, cfg.Runtime.EventSinks, 1)
			assert.Len(t, cfg.Runtime.TestWraps, 1)
			cfg.Event(event)
			_ = trace.Snapshot()
			cfg.Event(event)
		})
	}
	workers.Wait()
	records := trace.Snapshot()
	require.Len(t, records, 16)
	seen := make(map[string]bool)
	for _, record := range records {
		assert.NotContains(t, seen, record.Case.Name)
		seen[record.Case.Name] = true
		assert.Equal(t, []axiom.Event{event, event}, record.Events)
	}
}
