package teststats

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func readRunAttempts(dir, runID, processID string) ([]attemptArtifact, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var attempts []attemptArtifact
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "attempt-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		record, err := decodeAttemptArtifact(data)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if record.RunID != runID || record.ProcessID != processID {
			return nil, fmt.Errorf("read %s: attempt belongs to a different run or process", path)
		}
		if entry.Name() != fmt.Sprintf("attempt-%d.json", record.Number) {
			return nil, fmt.Errorf("read %s: filename does not match attempt number %d", path, record.Number)
		}
		attempts = append(attempts, record)
	}
	slices.SortFunc(attempts, func(a, b attemptArtifact) int { return cmp.Compare(a.Number, b.Number) })
	return attempts, nil
}

func decodeAttemptArtifact(data []byte) (attemptArtifact, error) {
	var record attemptArtifact
	if err := json.Unmarshal(data, &record); err != nil {
		return attemptArtifact{}, err
	}
	if err := record.validate(); err != nil {
		return attemptArtifact{}, err
	}
	return record, nil
}

// validate checks the fields used to rebuild a run from saved attempts.
func (a attemptArtifact) validate() error {
	if a.SchemaVersion != schemaVersion {
		return fmt.Errorf("unsupported schema version %d", a.SchemaVersion)
	}
	if a.Kind != "attempt" {
		return fmt.Errorf("expected attempt kind, got %q", a.Kind)
	}
	if a.Number < 1 {
		return errors.New("attempt number must be positive")
	}
	if a.Revision < 1 {
		return errors.New("attempt revision must be positive")
	}
	if a.Start.IsZero() {
		return errors.New("attempt start time is missing")
	}
	if a.Duration < 0 {
		return errors.New("attempt duration must not be negative")
	}

	switch a.State {
	case stateRunning:
		if a.Status != "" || !a.End.IsZero() || a.Duration != 0 {
			return errors.New("running attempt must not have an outcome or completed timing")
		}
	case stateFinished, stateFinalized:
		if a.End.IsZero() {
			return errors.New("completed attempt finish time is missing")
		}
		switch a.Status {
		case StatusPassed, StatusFailed, StatusSkipped:
		default:
			return fmt.Errorf("unknown attempt status %q", a.Status)
		}
	default:
		return fmt.Errorf("unknown attempt state %q", a.State)
	}
	return nil
}
