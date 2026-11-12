package teststats

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func finalizedArtifact(number int, status Status) attemptArtifact {
	record := completedArtifact(number, status)
	record.State = StateFinalized
	record.Revision = 3
	return record
}

func saveLoadAttempt(t *testing.T, dir string, record attemptArtifact) string {
	t.Helper()
	name := path.Join("processes", record.ProcessID, "runs", hex.EncodeToString([]byte(record.RunID)), fmt.Sprintf("attempt-%d.json", record.Number))
	require.NoError(t, writeArtifact(filepath.Join(dir, filepath.FromSlash(name)), record))
	return name
}

func TestLoad_SeparatesProcessesAndRebuildsRuns(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "results[1]")
	failed := finalizedArtifact(1, StatusFailed)
	failed.ProcessID = "process-1"
	firstPath := saveLoadAttempt(t, dir, failed)
	passed := finalizedArtifact(2, StatusPassed)
	passed.ProcessID = failed.ProcessID
	passed.Meta.Owner = "qa"
	saveLoadAttempt(t, dir, passed)
	other := finalizedArtifact(1, StatusPassed)
	other.ProcessID = "process-2"
	saveLoadAttempt(t, dir, other)
	skipped := finalizedArtifact(1, StatusSkipped)
	skipped.ProcessID = other.ProcessID
	skipped.RunID = "../../arbitrary/[run]"
	saveLoadAttempt(t, dir, skipped)

	// Cached summaries and temporary/unrelated files are not input to Load.
	runPath := filepath.Join(dir, filepath.Dir(firstPath), "run.json")
	require.NoError(t, os.WriteFile(runPath, []byte("invalid cached JSON"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(runPath), ".teststats-crash.tmp"), []byte("{"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "attempt-1.json"), []byte("unrelated"), 0600))

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.False(t, loaded.Incomplete())
	assert.Empty(t, loaded.Issues)
	assert.Empty(t, loaded.Unfinished)
	assert.Equal(t, Summary{
		Runs: Counts{
			Total:   3,
			Passed:  1,
			Skipped: 1,
			Flaky:   1,
		},
		Attempts: Counts{
			Total:   4,
			Passed:  2,
			Failed:  1,
			Skipped: 1,
		},
	}, loaded.Stats.Summary())
	runs := loaded.Stats.Runs()
	require.Len(t, runs, 3)
	assert.Equal(t, failed.RunID, runs[0].ID, "the original run ID is preserved")
	assert.Equal(t, failed.ProcessID, runs[0].ProcessID)
	assert.Equal(t, StatusFlaky, runs[0].Status)
	assert.Equal(t, "qa", runs[0].Meta.Owner)
	assert.Equal(t, failed.Duration+passed.Duration, runs[0].Duration)
	assert.Equal(t, passed.End.Sub(failed.Start), runs[0].Elapsed)
	assert.Equal(t, passed.End.Sub(failed.Start), loaded.Stats.Elapsed())

	again, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, runs, again.Stats.Runs(), "loading order is deterministic")
	cache, err := os.ReadFile(runPath)
	require.NoError(t, err)
	assert.Equal(t, "invalid cached JSON", string(cache), "loading must not rewrite files")
}

func TestLoad_UnfinishedAttemptsKeepTheirObservedState(t *testing.T) {
	dir := t.TempDir()
	first := finalizedArtifact(1, StatusPassed)
	saveLoadAttempt(t, dir, first)
	finished := completedArtifact(2, StatusFailed)
	finished.Meta.Tags = []string{"smoke"}
	saveLoadAttempt(t, dir, finished)
	running := newAttemptArtifact(first.RunID, Attempt{
		Number: 3,
		Start:  finished.End.Add(time.Second),
	}, StateRunning, 1)
	saveLoadAttempt(t, dir, running)

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.True(t, loaded.Incomplete())
	assert.Empty(t, loaded.Issues)
	assert.Equal(t, 2, loaded.Stats.Summary().Attempts.Total)
	assert.Equal(t, StatusFailed, loaded.Stats.Runs()[0].Status)
	assert.Equal(t, finished.End.Sub(first.Start), loaded.Stats.Elapsed(), "running attempts do not extend elapsed time")
	require.Len(t, loaded.Unfinished, 2)
	assert.Equal(t, StateFinished, loaded.Unfinished[0].State)
	assert.Equal(t, StatusFailed, loaded.Unfinished[0].Attempt.Status)
	assert.Equal(t, StateRunning, loaded.Unfinished[1].State)
	assert.Empty(t, loaded.Unfinished[1].Attempt.Status)
	assert.True(t, loaded.Unfinished[1].Attempt.End.IsZero())
	loaded.Unfinished[0].Attempt.Meta.Tags[0] = "changed"
	assert.Equal(t, []string{"smoke"}, loaded.Stats.Attempts()[1].Meta.Tags)
}

func TestLoad_InvalidAttemptsDoNotDiscardOtherResults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*attemptArtifact)
	}{
		{
			name:   "unsupported-version",
			change: func(a *attemptArtifact) { a.SchemaVersion++ },
		},
		{
			name:   "wrong-kind",
			change: func(a *attemptArtifact) { a.Kind = "run" },
		},
		{
			name:   "wrong-process",
			change: func(a *attemptArtifact) { a.ProcessID = "elsewhere" },
		},
		{
			name:   "wrong-run",
			change: func(a *attemptArtifact) { a.RunID = "elsewhere" },
		},
		{
			name:   "wrong-number",
			change: func(a *attemptArtifact) { a.Number = 2 },
		},
		{
			name:   "unknown-state",
			change: func(a *attemptArtifact) { a.State = "unknown" },
		},
		{
			name:   "unknown-status",
			change: func(a *attemptArtifact) { a.Status = "broken" },
		},
		{
			name:   "missing-finish",
			change: func(a *attemptArtifact) { a.End = time.Time{} },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			bad := finalizedArtifact(1, StatusFailed)
			name := saveLoadAttempt(t, dir, bad)
			tc.change(&bad)
			require.NoError(t, writeArtifact(filepath.Join(dir, name), bad))
			saveLoadAttempt(t, dir, finalizedArtifact(2, StatusPassed))

			loaded, err := Load(dir)
			require.NoError(t, err)
			assert.True(t, loaded.Incomplete())
			assert.Equal(t, 1, loaded.Stats.Summary().Attempts.Passed)
			assert.Equal(t, time.Millisecond, loaded.Stats.Elapsed(), "invalid attempts do not contribute timing")
			require.Len(t, loaded.Issues, 1)
			assert.Equal(t, name, loaded.Issues[0].Path)
			assert.Contains(t, loaded.Issues[0].Error(), name)
		})
	}
}

func TestLoad_MissingAndMalformedData(t *testing.T) {
	for _, payload := range []string{"{", "null", "[]", "{}", "{} {}"} {
		t.Run(payload, func(t *testing.T) {
			dir := t.TempDir()
			name := saveLoadAttempt(t, dir, finalizedArtifact(1, StatusPassed))
			require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(payload), 0600))
			loaded, err := Load(dir)
			require.NoError(t, err)
			assert.True(t, loaded.Incomplete())
			assert.Empty(t, loaded.Stats.Attempts())
			require.Len(t, loaded.Issues, 1)
		})
	}

	t.Run("empty-directory", func(t *testing.T) {
		loaded, err := Load(t.TempDir())
		require.NoError(t, err)
		assert.True(t, loaded.Incomplete())
		assert.Empty(t, loaded.Issues)
		assert.Empty(t, loaded.Stats.Attempts())
		assert.Zero(t, loaded.Stats.Elapsed())
	})
	t.Run("unreadable-root", func(t *testing.T) {
		for _, dir := range []string{"", "  ", filepath.Join(t.TempDir(), "missing")} {
			loaded, err := Load(dir)
			require.Error(t, err)
			assert.Nil(t, loaded)
		}
		file := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(file, nil, 0600))
		_, err := Load(file)
		require.Error(t, err)
	})
	t.Run("missing-attempts", func(t *testing.T) {
		dir := t.TempDir()
		for _, number := range []int{3, 1, 9} {
			saveLoadAttempt(t, dir, finalizedArtifact(number, StatusPassed))
		}
		loaded, err := Load(dir)
		require.NoError(t, err)
		assert.True(t, loaded.Incomplete())
		assert.Equal(t, 3, loaded.Stats.Summary().Attempts.Total)
		require.Len(t, loaded.Issues, 2)
		assert.Contains(t, loaded.Issues[0].Error(), "missing attempts 2 through 2")
		assert.Contains(t, loaded.Issues[1].Error(), "missing attempts 4 through 8")
	})
	t.Run("large-gap", func(t *testing.T) {
		dir := t.TempDir()
		record := finalizedArtifact(1, StatusPassed)
		record.Number = int(^uint(0) >> 1)
		saveLoadAttempt(t, dir, record)
		loaded, err := Load(dir)
		require.NoError(t, err)
		assert.True(t, loaded.Incomplete())
		require.Len(t, loaded.Issues, 1, "a gap is reported as a range, not expanded into billions of entries")
	})
}

func TestLoad_ValidatesArtifactLocation(t *testing.T) {
	for _, relative := range []string{
		"processes/process-1/runs/not-hex/attempt-1.json",
		"processes/process-1/runs/72756E2D31/attempt-1.json",
		"processes/process-1/runs/72756e2d31/attempt-01.json",
		"processes/process-1/runs/72756e2d31/attempt-0.json",
	} {
		t.Run(relative, func(t *testing.T) {
			dir := t.TempDir()
			record := finalizedArtifact(1, StatusPassed)
			record.ProcessID = "process-1"
			require.NoError(t, writeArtifact(filepath.Join(dir, relative), record))
			loaded, err := Load(dir)
			require.NoError(t, err)
			assert.True(t, loaded.Incomplete())
			assert.Empty(t, loaded.Stats.Attempts())
			assert.NotEmpty(t, loaded.Issues)
		})
	}
}

type unreadableFS struct {
	fs.FS
	name string
}

func (f unreadableFS) Open(name string) (fs.File, error) {
	if name == f.name {
		return nil, fs.ErrPermission
	}
	return f.FS.Open(name)
}

func TestLoad_DirectoryAndFileErrorsRetainOtherProcesses(t *testing.T) {
	files := fstest.MapFS{}
	for _, process := range []string{"one", "two"} {
		record := finalizedArtifact(1, StatusPassed)
		record.ProcessID = process
		data, err := json.Marshal(record)
		require.NoError(t, err)
		name := path.Join("processes", process, "runs", "72756e2d31", "attempt-1.json")
		files[name] = &fstest.MapFile{Data: data}
	}
	for _, name := range []string{
		".",
		"processes",
		"processes/one",
		"processes/one/runs",
		"processes/one/runs/72756e2d31",
		"processes/one/runs/72756e2d31/attempt-1.json",
	} {
		t.Run(name, func(t *testing.T) {
			loaded, err := loadFiles(unreadableFS{
				FS:   files,
				name: name,
			})
			if name == "." || name == "processes" {
				require.ErrorIs(t, err, fs.ErrPermission)
				assert.Nil(t, loaded)
				return
			}
			require.NoError(t, err)
			assert.True(t, loaded.Incomplete())
			assert.Equal(t, 1, loaded.Stats.Summary().Attempts.Passed)
			require.Len(t, loaded.Issues, 1)
			assert.ErrorIs(t, loaded.Issues[0], fs.ErrPermission)
		})
	}
}

func TestLoad_DoesNotFollowSymbolicLinks(t *testing.T) {
	for _, name := range []string{
		"processes",
		"processes/one",
		"processes/one/runs",
		"processes/one/runs/72756e2d31",
		"processes/one/runs/72756e2d31/attempt-1.json",
	} {
		t.Run(name, func(t *testing.T) {
			files := fstest.MapFS{
				name: &fstest.MapFile{
					Mode: fs.ModeSymlink,
					Data: []byte("target"),
				},
			}
			loaded, err := loadFiles(files)
			if name == "processes" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, loaded.Incomplete())
			assert.Empty(t, loaded.Stats.Attempts())
			assert.NotEmpty(t, loaded.Issues)
		})
	}
}

func TestLoad_EmptyProcessAndRunDirectoriesAreIncomplete(t *testing.T) {
	for _, name := range []string{"processes/one", "processes/one/runs", "processes/one/runs/72756e2d31"} {
		t.Run(name, func(t *testing.T) {
			files := fstest.MapFS{
				name: &fstest.MapFile{Mode: fs.ModeDir},
			}
			loaded, err := loadFiles(files)
			require.NoError(t, err)
			assert.True(t, loaded.Incomplete())
			assert.NotEmpty(t, loaded.Issues)
		})
	}
}
