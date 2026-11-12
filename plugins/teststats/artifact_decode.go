package teststats

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

func decodeAttemptArtifact(data []byte) (attemptArtifact, error) {
	if err := validateAttemptFields(data); err != nil {
		return attemptArtifact{}, err
	}
	var record attemptArtifact
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return attemptArtifact{}, err
	}
	if err := record.validate(); err != nil {
		return attemptArtifact{}, err
	}
	return record, nil
}

// JSON null and absent fields must not become valid-looking Go zero values.
func validateAttemptFields(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	required := []string{
		"schema_version",
		"kind",
		"process_id",
		"run_id",
		"number",
		"revision",
		"state",
		"name",
		"test_name",
		"meta",
		"started_at",
		"duration_ns",
	}
	for _, name := range required {
		if _, exists := fields[name]; !exists {
			return fmt.Errorf("required field %q is missing", name)
		}
	}
	for name := range fields {
		switch name {
		case "case_id", "status", "skip_reason", "error", "finished_at":
			continue
		}
		if !slices.Contains(required, name) {
			return fmt.Errorf("unknown attempt field %q", name)
		}
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(fields["meta"], &meta); err != nil {
		return fmt.Errorf("meta: %w", err)
	}
	// The Go decoder also accepts capitalized field names; schema v1 does not.
	for name := range meta {
		if name != strings.ToLower(name) {
			return fmt.Errorf("unknown metadata field %q", name)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := validateJSONValue(decoder); err != nil {
		return err
	}
	var state AttemptState
	if err := json.Unmarshal(fields["state"], &state); err != nil {
		return err
	}
	if state == StateRunning {
		if _, exists := fields["status"]; exists {
			return errors.New("running attempt must omit status")
		}
		if _, exists := fields["finished_at"]; exists {
			return errors.New("running attempt must omit finished_at")
		}
	}
	return nil
}

// Attempt documents contain neither null values nor duplicate object keys.
func validateJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("attempt fields must not contain JSON null")
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return nil
	}
	keys := make(map[string]bool)
	for decoder.More() {
		if delimiter == '{' {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			// JSON object keys are strings; syntax was checked by Unmarshal.
			name := key.(string)
			if keys[name] {
				return fmt.Errorf("duplicate JSON key %q", name)
			}
			keys[name] = true
		}
		if err := validateJSONValue(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

// validate checks the fields used to rebuild a run from saved attempts.
func (a attemptArtifact) validate() error {
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

	switch a.State {
	case StateRunning:
		if a.Status != "" || !a.End.IsZero() || a.Duration != 0 {
			return errors.New("running attempt must not have an outcome or completed timing")
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
