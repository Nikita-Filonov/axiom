package teststats

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func artifactFields(t *testing.T, record attemptArtifact) map[string]json.RawMessage {
	t.Helper()
	data, err := json.Marshal(record)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &fields))
	return fields
}

func TestDecodeAttempt_RejectsMalformedJSON(t *testing.T) {
	for _, payload := range []string{
		"",
		"null",
		"[]",
		"true",
		"{} {}",
		`{"meta":`,
		`{"meta":tru}`,
		`{true:"value"}`,
		`{"meta":{]}`,
		`{"meta":{"tags":["smoke",]}}`,
	} {
		t.Run(payload, func(t *testing.T) {
			record, err := decodeAttemptArtifact([]byte(payload))
			require.Error(t, err)
			assert.Empty(t, record)
		})
	}
}

func TestDecodeAttempt_RequiresFields(t *testing.T) {
	for _, field := range []string{
		"schema_version", "kind", "process_id", "run_id", "number", "revision", "state",
		"name", "test_name", "meta", "started_at", "duration_ns", "status", "finished_at",
	} {
		t.Run(field, func(t *testing.T) {
			fields := artifactFields(t, finalizedArtifact(1, StatusPassed))
			delete(fields, field)
			data, err := json.Marshal(fields)
			require.NoError(t, err)
			_, err = decodeAttemptArtifact(data)
			require.Error(t, err, "missing %s must not become a valid zero value", field)
		})
	}
}

func TestDecodeAttempt_RejectsInvalidFields(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		value string
		want  string
	}{
		{
			name:  "metadata-array",
			field: "meta",
			value: `[]`,
			want:  "meta:",
		},
		{
			name:  "metadata-null",
			field: "meta",
			value: `null`,
			want:  "null",
		},
		{
			name:  "null-tag",
			field: "meta",
			value: `{"tags":[null]}`,
			want:  "null",
		},
		{
			name:  "null-label",
			field: "meta",
			value: `{"labels":{"team":null}}`,
			want:  "null",
		},
		{
			name:  "duplicate-label",
			field: "meta",
			value: `{"labels":{"team":"one","team":"two"}}`,
			want:  `duplicate JSON key "team"`,
		},
		{
			name:  "unknown-metadata-field",
			field: "meta",
			value: `{"unknown":"value"}`,
			want:  "unknown field",
		},
		{
			name:  "capitalized-metadata-field",
			field: "meta",
			value: `{"Owner":"qa"}`,
			want:  "unknown metadata field",
		},
		{
			name:  "unknown-attempt-field",
			field: "unknown",
			value: `true`,
			want:  "unknown attempt field",
		},
		{
			name:  "capitalized-status",
			field: "STATUS",
			value: `"failed"`,
			want:  "unknown attempt field",
		},
		{
			name:  "null-name",
			field: "name",
			value: `null`,
			want:  "null",
		},
		{
			name:  "non-string-state",
			field: "state",
			value: `123`,
			want:  "cannot unmarshal number",
		},
		{
			name:  "empty-process",
			field: "process_id",
			value: `""`,
			want:  "IDs must not be empty",
		},
		{
			name:  "empty-run",
			field: "run_id",
			value: `""`,
			want:  "IDs must not be empty",
		},
		{
			name:  "zero-number",
			field: "number",
			value: `0`,
			want:  "attempt number must be positive",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := artifactFields(t, finalizedArtifact(1, StatusPassed))
			fields[tc.field] = json.RawMessage(tc.value)
			data, err := json.Marshal(fields)
			require.NoError(t, err)
			_, err = decodeAttemptArtifact(data)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestDecodeAttempt_RejectsDuplicateStatus(t *testing.T) {
	valid, err := json.Marshal(finalizedArtifact(1, StatusPassed))
	require.NoError(t, err)
	payload := strings.TrimSuffix(string(valid), "}") + `,"status":"failed"}`
	_, err = decodeAttemptArtifact([]byte(payload))
	require.ErrorContains(t, err, `duplicate JSON key "status"`)
}

func TestDecodeAttempt_RunningAttemptCannotHaveCompletedTimingOrOutcome(t *testing.T) {
	for _, tc := range []struct {
		field string
		value string
		want  string
	}{
		{
			field: "status",
			value: `""`,
			want:  "must omit status",
		},
		{
			field: "finished_at",
			value: `"0001-01-01T00:00:00Z"`,
			want:  "must omit finished_at",
		},
		{
			field: "duration_ns",
			value: `1`,
			want:  "duration must be zero",
		},
	} {
		t.Run(tc.field, func(t *testing.T) {
			running := newAttemptArtifact("run", Attempt{
				Number: 1,
				Start:  time.Now(),
			}, StateRunning, 1)
			fields := artifactFields(t, running)
			fields[tc.field] = json.RawMessage(tc.value)
			data, err := json.Marshal(fields)
			require.NoError(t, err)
			_, err = decodeAttemptArtifact(data)
			require.ErrorContains(t, err, tc.want)
		})
	}
}
