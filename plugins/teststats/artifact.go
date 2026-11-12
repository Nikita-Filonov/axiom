package teststats

import (
	"crypto/rand"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

const schemaVersion = 1

// A namespace shared by this process's plugin instances. A PID alone can be
// reused, including by different containers writing to the same directory.
var processID = rand.Text()

// AttemptState describes how far an exported attempt reached in its lifecycle.
// It is separate from Status, which describes the test outcome.
type AttemptState string

const (
	// StateRunning has no completed outcome yet.
	StateRunning AttemptState = "running"
	// StateFinished has an outcome that has not been rechecked during parent cleanup.
	StateFinished AttemptState = "finished"
	// StateFinalized has been rechecked during parent cleanup.
	StateFinalized AttemptState = "finalized"
)

// Artifact types deliberately separate the versioned wire format from the
// in-memory API. Adding a field to Attempt or axiom.Meta does not change JSON.
type attemptArtifact struct {
	SchemaVersion int           `json:"schema_version"`
	Kind          string        `json:"kind"`
	ProcessID     string        `json:"process_id"`
	RunID         string        `json:"run_id"`
	Number        int           `json:"number"`
	Revision      uint64        `json:"revision"`
	State         AttemptState  `json:"state"`
	CaseID        string        `json:"case_id,omitempty"`
	Name          string        `json:"name"`
	TestName      string        `json:"test_name"`
	Meta          artifactMeta  `json:"meta"`
	Status        Status        `json:"status,omitempty"`
	SkipReason    string        `json:"skip_reason,omitempty"`
	Error         string        `json:"error,omitempty"`
	Start         time.Time     `json:"started_at"`
	End           time.Time     `json:"finished_at,omitzero"`
	Duration      time.Duration `json:"duration_ns"`
}

type runArtifact struct {
	SchemaVersion int                `json:"schema_version"`
	Kind          string             `json:"kind"`
	ProcessID     string             `json:"process_id"`
	RunID         string             `json:"run_id"`
	Attempts      []attemptReference `json:"attempts"`
	Summary       *runSummary        `json:"summary"`
}

type attemptReference struct {
	Number   int          `json:"number"`
	Revision uint64       `json:"revision"`
	State    AttemptState `json:"state"`
}

type runSummary struct {
	CaseID   string        `json:"case_id,omitempty"`
	Name     string        `json:"name"`
	Meta     artifactMeta  `json:"meta"`
	Status   Status        `json:"status"`
	Start    time.Time     `json:"started_at"`
	End      time.Time     `json:"finished_at"`
	Duration time.Duration `json:"duration_ns"`
	Elapsed  time.Duration `json:"elapsed_ns"`
}

func newAttemptArtifact(runID string, attempt Attempt, state AttemptState, revision uint64) attemptArtifact {
	return attemptArtifact{
		SchemaVersion: schemaVersion,
		Kind:          "attempt",
		ProcessID:     processID,
		RunID:         runID,
		Number:        attempt.Number,
		Revision:      revision,
		State:         state,
		CaseID:        attempt.CaseID,
		Name:          attempt.Name,
		TestName:      attempt.TestName,
		Meta:          newArtifactMeta(attempt.Meta),
		Status:        attempt.Status,
		SkipReason:    attempt.SkipReason,
		Error:         attempt.Error,
		Start:         attempt.Start.UTC(),
		End:           attempt.End.UTC(),
		Duration:      attempt.Duration,
	}
}

func (a attemptArtifact) toAttempt() Attempt {
	return Attempt{
		ProcessID:  a.ProcessID,
		RunID:      a.RunID,
		Number:     a.Number,
		CaseID:     a.CaseID,
		Name:       a.Name,
		TestName:   a.TestName,
		Meta:       a.Meta.toMeta(),
		Status:     a.Status,
		SkipReason: a.SkipReason,
		Error:      a.Error,
		Start:      a.Start,
		End:        a.End,
		Duration:   a.Duration,
	}
}

func newRunArtifact(runID string, records []attemptArtifact) runArtifact {
	references := make([]attemptReference, 0, len(records))
	var completed []Attempt
	for _, record := range records {
		references = append(references, attemptReference{
			Number:   record.Number,
			Revision: record.Revision,
			State:    record.State,
		})
		if record.State != StateRunning {
			completed = append(completed, record.toAttempt())
		}
	}
	return runArtifact{
		SchemaVersion: schemaVersion,
		Kind:          "run",
		ProcessID:     processID,
		RunID:         runID,
		Attempts:      references,
		Summary:       newRunSummary(completed),
	}
}

func newRunSummary(attempts []Attempt) *runSummary {
	if len(attempts) == 0 {
		return nil
	}
	// Reuse the same metadata selection, retry classification, and timing as Stats.Runs().
	run := newRun(attempts)
	return &runSummary{
		CaseID:   run.CaseID,
		Name:     run.Name,
		Meta:     newArtifactMeta(run.Meta),
		Status:   run.Status,
		Start:    run.Start,
		End:      run.End,
		Duration: run.Duration,
		Elapsed:  run.Elapsed,
	}
}

type artifactMeta struct {
	Epic        string            `json:"epic,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Suite       string            `json:"suite,omitempty"`
	Story       string            `json:"story,omitempty"`
	Layer       string            `json:"layer,omitempty"`
	Issues      []string          `json:"issues,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Feature     string            `json:"feature,omitempty"`
	Platform    string            `json:"platform,omitempty"`
	Severity    axiom.Severity    `json:"severity,omitempty"`
	SubSuite    string            `json:"sub_suite,omitempty"`
	TestCases   []string          `json:"test_cases,omitempty"`
	ParentSuite string            `json:"parent_suite,omitempty"`
}

func newArtifactMeta(meta axiom.Meta) artifactMeta {
	meta = meta.Copy()
	return artifactMeta{
		Epic:        meta.Epic,
		Tags:        meta.Tags,
		Owner:       meta.Owner,
		Suite:       meta.Suite,
		Story:       meta.Story,
		Layer:       meta.Layer,
		Issues:      meta.Issues,
		Labels:      meta.Labels,
		Feature:     meta.Feature,
		Platform:    meta.Platform,
		Severity:    meta.Severity,
		SubSuite:    meta.SubSuite,
		TestCases:   meta.TestCases,
		ParentSuite: meta.ParentSuite,
	}
}

func (m artifactMeta) toMeta() axiom.Meta {
	meta := axiom.Meta{
		Epic:        m.Epic,
		Tags:        m.Tags,
		Owner:       m.Owner,
		Suite:       m.Suite,
		Story:       m.Story,
		Layer:       m.Layer,
		Issues:      m.Issues,
		Labels:      m.Labels,
		Feature:     m.Feature,
		Platform:    m.Platform,
		Severity:    m.Severity,
		SubSuite:    m.SubSuite,
		TestCases:   m.TestCases,
		ParentSuite: m.ParentSuite,
	}
	return meta.Copy()
}
