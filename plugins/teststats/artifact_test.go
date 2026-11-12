package teststats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifact_V1Contract(t *testing.T) {
	start := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	a := newAttemptArtifact("run-1", Attempt{
		Number:   1,
		CaseID:   "C-1",
		Name:     "login",
		TestName: "TestLogin/login",
		Meta: axiom.Meta{
			Owner:  "qa",
			Tags:   []string{"smoke"},
			Labels: map[string]string{"component": "login"},
		},
		Status:   StatusPassed,
		Start:    start,
		End:      start.Add(time.Second),
		Duration: time.Second,
	}, stateFinalized, 3)
	run := newRunArtifact(a.RunID, []attemptArtifact{a})
	// Process identity varies per binary; the rest of the format is stable.
	a.ProcessID, run.ProcessID = "process-1", "process-1"
	for name, record := range map[string]any{"attempt-v1.json": a, "run-v1.json": run} {
		t.Run(name, func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join("testdata", name))
			require.NoError(t, err)
			actual, err := json.Marshal(record)
			require.NoError(t, err)
			assert.JSONEq(t, string(expected), string(actual))
		})
	}
}

func TestArtifact_RunningAttemptDoesNotInventOutcome(t *testing.T) {
	a := newAttemptArtifact("run-1", Attempt{
		Number: 1,
		Start:  time.Now(),
	}, stateRunning, 1)
	data, err := json.Marshal(a)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fields))
	assert.NotContains(t, fields, "status")
	assert.NotContains(t, fields, "finished_at")
	assert.Nil(t, newRunArtifact(a.RunID, []attemptArtifact{a}).Summary)
	prior := completedArtifact(1, StatusFailed)
	a.Number = 2
	run := newRunArtifact(a.RunID, []attemptArtifact{prior, a})
	require.NotNil(t, run.Summary)
	assert.Equal(t, StatusFailed, run.Summary.Status)
	assert.Equal(t, prior.Duration, run.Summary.Duration)
	assert.Equal(t, stateRunning, run.Attempts[1].State)
}

func TestArtifact_MetaIsSnapshot(t *testing.T) {
	meta := axiom.Meta{
		Tags:   []string{"original"},
		Labels: map[string]string{"key": "original"},
	}
	a := newAttemptArtifact("run", Attempt{Meta: meta}, stateRunning, 1)
	meta.Tags[0], meta.Labels["key"] = "changed", "changed"
	assert.Equal(t, []string{"original"}, a.Meta.Tags)
	assert.Equal(t, "original", a.Meta.Labels["key"])
}

func TestArtifact_RunSummaryUsesLastCompletedAttempt(t *testing.T) {
	meta := axiom.Meta{
		Epic:        "authentication",
		Tags:        []string{"smoke"},
		Owner:       "qa",
		Suite:       "backend",
		Story:       "login",
		Layer:       "api",
		Issues:      []string{"ISSUE-1"},
		Labels:      map[string]string{"component": "login"},
		Feature:     "sessions",
		Platform:    "web",
		Severity:    axiom.SeverityCritical,
		SubSuite:    "positive",
		TestCases:   []string{"C-1"},
		ParentSuite: "integration",
	}
	first := completedArtifact(1, StatusFailed)
	second := completedArtifact(2, StatusPassed)
	second.Meta = newArtifactMeta(meta)
	running := newAttemptArtifact("run-1", Attempt{
		Number: 3,
		Start:  time.Now(),
	}, stateRunning, 1)

	run := newRunArtifact("run-1", []attemptArtifact{first, second, running})

	require.NotNil(t, run.Summary)
	assert.Equal(t, StatusFlaky, run.Summary.Status)
	assert.Equal(t, meta, run.Summary.Meta.toMeta())
	assert.Equal(t, first.Duration+second.Duration, run.Summary.Duration)
}
