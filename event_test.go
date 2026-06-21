package axiom_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent_NormalizesTimeAndUsesTypeField(t *testing.T) {
	e := axiom.NewEvent(
		axiom.EventTypeStepStart,
		axiom.WithEventName("step"),
		axiom.WithEventMessage("hello"),
	)

	assert.Equal(t, axiom.EventTypeStepStart, e.Type)
	assert.Equal(t, "step", e.Name)
	assert.Equal(t, "hello", e.Message)
	require.NotEmpty(t, e.Time)

	_, err := time.Parse(time.RFC3339Nano, e.Time)
	require.NoError(t, err)

	data, err := json.Marshal(e)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"type":"step.start"`)
	assert.Contains(t, string(data), `"message":"hello"`)
	assert.NotContains(t, string(data), `"event"`)
	assert.NotContains(t, string(data), `"status"`)
	assert.NotContains(t, string(data), `"case"`)
	assert.NotContains(t, string(data), `"attempt"`)
}

func TestNewEvent_UsesExplicitTime(t *testing.T) {
	e := axiom.NewEvent(
		axiom.EventTypeLog,
		axiom.WithEventTime("2026-06-21T10:00:00Z"),
	)

	assert.Equal(t, "2026-06-21T10:00:00Z", e.Time)
}

func TestWithEventMessageFormatsAny(t *testing.T) {
	e := axiom.NewEvent(
		axiom.EventTypeLog,
		axiom.WithEventMessage(fmt.Errorf("boom")),
	)

	assert.Equal(t, "boom", e.Message)
}

func TestEventTypeString(t *testing.T) {
	assert.Equal(t, "step.start", axiom.EventTypeStepStart.String())
}

func TestEventTypeValues(t *testing.T) {
	cases := map[axiom.EventType]string{
		axiom.EventTypeRunnerBeforeAllStart:  "runner.before-all.start",
		axiom.EventTypeRunnerBeforeAllFinish: "runner.before-all.finish",
		axiom.EventTypeRunnerBeforeAllPanic:  "runner.before-all.panic",
		axiom.EventTypeRunnerAfterAllStart:   "runner.after-all.start",
		axiom.EventTypeRunnerAfterAllFinish:  "runner.after-all.finish",
		axiom.EventTypeRunnerAfterAllPanic:   "runner.after-all.panic",
		axiom.EventTypeCaseStart:             "case.start",
		axiom.EventTypeCaseFinish:            "case.finish",
		axiom.EventTypeCasePanic:             "case.panic",
		axiom.EventTypeStepStart:             "step.start",
		axiom.EventTypeStepFinish:            "step.finish",
		axiom.EventTypeStepPanic:             "step.panic",
		axiom.EventTypeSetupStart:            "setup.start",
		axiom.EventTypeSetupFinish:           "setup.finish",
		axiom.EventTypeSetupPanic:            "setup.panic",
		axiom.EventTypeTeardownStart:         "teardown.start",
		axiom.EventTypeTeardownFinish:        "teardown.finish",
		axiom.EventTypeTeardownPanic:         "teardown.panic",
		axiom.EventTypeFixtureSetupStart:     "fixture.setup.start",
		axiom.EventTypeFixtureSetupFinish:    "fixture.setup.finish",
		axiom.EventTypeFixtureSetupFailed:    "fixture.setup.failed",
		axiom.EventTypeFixtureCleanupStart:   "fixture.cleanup.start",
		axiom.EventTypeFixtureCleanupFinish:  "fixture.cleanup.finish",
		axiom.EventTypeFixtureCleanupPanic:   "fixture.cleanup.panic",
		axiom.EventTypeResourceSetupStart:    "resource.setup.start",
		axiom.EventTypeResourceSetupFinish:   "resource.setup.finish",
		axiom.EventTypeResourceSetupFailed:   "resource.setup.failed",
		axiom.EventTypeResourceCleanupStart:  "resource.cleanup.start",
		axiom.EventTypeResourceCleanupFinish: "resource.cleanup.finish",
		axiom.EventTypeResourceCleanupPanic:  "resource.cleanup.panic",
		axiom.EventTypeLog:                   "log",
		axiom.EventTypeAssert:                "assert",
		axiom.EventTypeArtefact:              "artefact",
	}

	for eventType, value := range cases {
		assert.Equal(t, value, eventType.String())
	}
}

func TestEventBuildersUseStringMethods(t *testing.T) {
	logEvent := axiom.NewLogEvent(axiom.NewInfoLog("hello"))
	assert.Equal(t, axiom.EventTypeLog, logEvent.Type)
	assert.Equal(t, axiom.LogLevelInfo.String(), logEvent.Name)
	assert.Equal(t, "hello", logEvent.Message)

	assertEvent := axiom.NewAssertEvent(axiom.NewEqualAssert(1, 1, "ok"))
	assert.Equal(t, axiom.EventTypeAssert, assertEvent.Type)
	assert.Equal(t, axiom.AssertEqual.String(), assertEvent.Name)
	assert.Equal(t, "ok", assertEvent.Message)

	artefactEvent := axiom.NewArtefactEvent(axiom.NewTextArtefact("report", "payload"))
	assert.Equal(t, axiom.EventTypeArtefact, artefactEvent.Type)
	assert.Equal(t, axiom.ArtefactTypeText.String(), artefactEvent.Name)
	assert.Equal(t, "report", artefactEvent.Message)
}

func requireEventTypes(t *testing.T, events []axiom.Event, types ...axiom.EventType) {
	t.Helper()
	require.Len(t, events, len(types))
	for i, eventType := range types {
		assert.Equal(t, eventType, events[i].Type)
	}
}
