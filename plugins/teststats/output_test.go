package teststats

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func completedArtifact(number int, status Status) attemptArtifact {
	start := time.Date(2026, 11, 11, 10, 0, number, 0, time.UTC)
	return newAttemptArtifact("run-1", Attempt{
		Number:   number,
		Name:     "case",
		Status:   status,
		Start:    start,
		End:      start.Add(time.Millisecond),
		Duration: time.Millisecond,
	}, stateFinished, 2)
}

func TestOutput_RunWriteFailureRetainsAttemptAndCanRecover(t *testing.T) {
	w := &outputWriter{dir: t.TempDir()}
	a := completedArtifact(1, StatusFailed)
	a.RunID = "../../arbitrary/[run]"
	dir := filepath.Join(w.dir, "processes", processID, "runs", hex.EncodeToString([]byte(a.RunID)))
	runPath := filepath.Join(dir, "run.json")
	require.NoError(t, os.MkdirAll(runPath, 0755))
	require.Error(t, w.write(a), "run.json is a directory")
	attempts, err := readRunAttempts(dir, a.RunID, processID)
	require.NoError(t, err)
	require.Len(t, attempts, 1, "failure to update run.json must not undo the attempt")
	assert.Equal(t, a.RunID, attempts[0].RunID)
	require.NoError(t, os.Remove(runPath))
	second := completedArtifact(2, StatusPassed)
	second.RunID = a.RunID
	// A different writer recovers the history from attempts already on disk.
	require.NoError(t, (&outputWriter{dir: w.dir}).write(second))
	data, err := os.ReadFile(runPath)
	require.NoError(t, err)
	var run runArtifact
	require.NoError(t, json.Unmarshal(data, &run))
	require.Len(t, run.Attempts, 2)
	require.NotNil(t, run.Summary)
	assert.Equal(t, StatusFlaky, run.Summary.Status)
}

func TestOutput_CorruptAttemptCannotProduceSuccessfulSummary(t *testing.T) {
	for _, corrupt := range []struct {
		name   string
		change func(*attemptArtifact)
	}{
		{
			name:   "schema",
			change: func(a *attemptArtifact) { a.SchemaVersion++ },
		},
		{
			name:   "identity",
			change: func(a *attemptArtifact) { a.RunID = "different" },
		},
		{
			name:   "number",
			change: func(a *attemptArtifact) { a.Number++ },
		},
		{
			name:   "revision",
			change: func(a *attemptArtifact) { a.Revision = 0 },
		},
		{
			name:   "state",
			change: func(a *attemptArtifact) { a.State = "unknown" },
		},
		{
			name:   "status",
			change: func(a *attemptArtifact) { a.Status = "broken" },
		},
		{
			name:   "start",
			change: func(a *attemptArtifact) { a.Start = time.Time{} },
		},
		{
			name:   "end",
			change: func(a *attemptArtifact) { a.End = time.Time{} },
		},
		{
			name:   "duration",
			change: func(a *attemptArtifact) { a.Duration = -1 },
		},
		{
			name:   "running-with-result",
			change: func(a *attemptArtifact) { a.State = stateRunning },
		},
		{
			name:   "invalid-json",
			change: nil,
		},
	} {
		t.Run(corrupt.name, func(t *testing.T) {
			w := &outputWriter{dir: t.TempDir()}
			a := completedArtifact(1, StatusFailed)
			require.NoError(t, w.write(a))
			dir := filepath.Join(w.dir, "processes", processID, "runs", hex.EncodeToString([]byte(a.RunID)))
			runPath := filepath.Join(dir, "run.json")
			previous, err := os.ReadFile(runPath)
			require.NoError(t, err)
			path := filepath.Join(dir, "attempt-1.json")
			if corrupt.change == nil {
				require.NoError(t, os.WriteFile(path, []byte("{"), 0600))
			} else {
				corrupt.change(&a)
				require.NoError(t, writeArtifact(path, a))
			}
			require.Error(t, w.write(completedArtifact(2, StatusPassed)))
			current, err := os.ReadFile(runPath)
			require.NoError(t, err)
			assert.Equal(t, string(previous), string(current), "a corrupt retry must not be dropped from the summary")
			require.FileExists(t, filepath.Join(dir, "attempt-2.json"))
		})
	}
}
