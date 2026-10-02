package testjunit

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedOutcome struct{ failed, skipped bool }

func (s *fixedOutcome) Failed() bool  { return s.failed }
func (s *fixedOutcome) Skipped() bool { return s.skipped }

func TestOutcome(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		failed, skipped, lifecycle bool
		want                       status
	}{
		{name: "passed", want: statusPassed},
		{name: "skipped", skipped: true, want: statusSkipped},
		{name: "failed", failed: true, want: statusFailed},
		{name: "failure wins over skip", failed: true, skipped: true, want: statusFailed},
		{name: "empty lifecycle error still fails", lifecycle: true, skipped: true, want: statusFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, outcome(&fixedOutcome{failed: tc.failed, skipped: tc.skipped}, tc.lifecycle))
		})
	}
}

func TestRecorderLifecycleFailures(t *testing.T) {
	for _, event := range []axiom.EventType{
		axiom.EventTypeCasePanic, axiom.EventTypeStepPanic,
		axiom.EventTypeSetupPanic, axiom.EventTypeTeardownPanic,
		axiom.EventTypeFixtureSetupFailed, axiom.EventTypeFixtureCleanupPanic,
	} {
		t.Run(string(event), func(t *testing.T) {
			reporter := NewReporter()
			t.Run("attempt", func(t *testing.T) {
				r := &recorder{cfg: &axiom.Config{SubT: t}, reporter: reporter}
				r.observe(axiom.Event{Type: event, Message: "before start"})
				r.observe(axiom.Event{Type: axiom.EventTypeCaseStart})
				r.observe(axiom.Event{Type: event, Message: "first error"})
				r.observe(axiom.Event{Type: event, Message: "second error"})
				r.observe(axiom.Event{Type: axiom.EventTypeCaseFinish})
				assert.Empty(t, reporter.snapshot(), "attempt published before cleanup")
			})
			got := reporter.snapshot()
			require.Len(t, got, 1)
			assert.Equal(t, statusFailed, got[0].Status)
			assert.Equal(t, "first error", got[0].Error)
			assert.GreaterOrEqual(t, got[0].Duration, time.Duration(0))
		})
	}
}

func TestRecorderEmptyErrorAndDuplicateEvents(t *testing.T) {
	reporter := NewReporter()
	r := &recorder{cfg: &axiom.Config{}, reporter: reporter}
	r.begin("")
	r.finish(&fixedOutcome{})
	assert.Nil(t, r.attempt, "recorded an attempt without testing.T")
	assert.Empty(t, reporter.snapshot())
	t.Run("attempt", func(t *testing.T) {
		r.cfg.SubT = t
		r.observe(axiom.Event{Type: axiom.EventTypeCaseStart})
		r.observe(axiom.Event{Type: axiom.EventTypeCaseSkip, Message: "duplicate"})
		r.fail("")
		r.fail("later error")
		r.finish(&fixedOutcome{})
		r.fail("after finish")
		r.finish(&fixedOutcome{})
	})
	got := reporter.snapshot()
	require.Len(t, got, 1)
	assert.Equal(t, statusFailed, got[0].Status)
	assert.Empty(t, got[0].Error)
	assert.Empty(t, got[0].SkipReason)
}

func TestRecorderDurationIncludesCleanup(t *testing.T) {
	reporter := NewReporter()
	var cleanupDuration time.Duration
	t.Run("attempt", func(t *testing.T) {
		r := &recorder{cfg: &axiom.Config{RootT: t}, reporter: reporter}
		r.begin("")
		t.Cleanup(func() {
			start := time.Now()
			time.Sleep(time.Millisecond)
			cleanupDuration = time.Since(start)
		})
	})
	got := reporter.snapshot()
	require.Len(t, got, 1)
	assert.GreaterOrEqual(t, got[0].Duration, cleanupDuration, "cleanup duration is missing")
}
