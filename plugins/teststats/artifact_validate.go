package teststats

import (
	"encoding/json"
	"errors"
	"fmt"
)

// validate checks the fields used to rebuild a run from saved attempts.
func (a attemptArtifact) validate(fields map[string]json.RawMessage) error {
	if a.SchemaVersion != schemaVersion {
		return fmt.Errorf("unsupported schema version %d", a.SchemaVersion)
	}
	if a.Kind != "attempt" {
		return fmt.Errorf("expected attempt kind, got %q", a.Kind)
	}
	if a.ProcessID == "" || a.RunID == "" {
		return errors.New("process and run IDs must not be empty")
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

	return a.validateOutcome(fields)
}

func (a attemptArtifact) validateOutcome(fields map[string]json.RawMessage) error {
	switch a.State {
	case StateRunning:
		for _, name := range []string{"status", "finished_at"} {
			if _, exists := fields[name]; exists {
				return fmt.Errorf("running attempt must omit %s", name)
			}
		}
		if a.Duration != 0 {
			return errors.New("running attempt duration must be zero")
		}
	case StateFinished, StateFinalized:
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
