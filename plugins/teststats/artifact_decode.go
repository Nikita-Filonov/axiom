package teststats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

func decodeAttemptArtifact(data []byte) (attemptArtifact, error) {
	fields, err := readAttemptFields(data)
	if err != nil {
		return attemptArtifact{}, err
	}
	var record attemptArtifact
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return attemptArtifact{}, err
	}
	if err := record.validate(fields); err != nil {
		return attemptArtifact{}, err
	}
	return record, nil
}

// JSON null and absent fields must not become valid-looking Go zero values.
func readAttemptFields(data []byte) (map[string]json.RawMessage, error) {
	if err := validateJSONValue(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if err := validateAttemptFields(fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func validateAttemptFields(fields map[string]json.RawMessage) error {
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
	return validateMetadataFields(fields["meta"])
}

func validateMetadataFields(data []byte) error {
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("meta: %w", err)
	}
	// The Go decoder also accepts capitalized field names; schema v1 does not.
	for name := range meta {
		if name != strings.ToLower(name) {
			return fmt.Errorf("unknown metadata field %q", name)
		}
	}
	return nil
}
