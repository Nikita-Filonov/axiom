package teststats

import (
	"cmp"
	"slices"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// Run groups the attempts of one RunCase invocation. CaseID, Name, and Meta
// come from the last attempt. Status is the last attempt's status, except that
// a pass after a failure is flaky and a skip after a failure stays failed.
// A flaky run is still a failed Go test; the plugin never changes that.
type Run struct {
	ID     string
	CaseID string
	Name   string
	Meta   axiom.Meta
	Status Status

	// Start and End span all attempts. Duration sums the attempt durations,
	// while Elapsed, End minus Start, also includes retry delays.
	Start    time.Time
	End      time.Time
	Duration time.Duration
	Elapsed  time.Duration

	// Attempts are ordered by Number.
	Attempts []Attempt
}

// Runs groups attempts by RunID, ordered by each run's earliest start.
// Attempts without a RunID are runs of their own.
func (s *Stats) Runs() []Run {
	var groups [][]Attempt
	index := make(map[string]int)
	for _, a := range s.Attempts() {
		i, ok := index[a.RunID]
		if !ok || a.RunID == "" {
			i = len(groups)
			index[a.RunID] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], a)
	}

	runs := make([]Run, len(groups))
	for i, attempts := range groups {
		runs[i] = newRun(attempts)
	}

	return runs
}

func newRun(attempts []Attempt) Run {
	slices.SortStableFunc(attempts, func(a, b Attempt) int { return cmp.Compare(a.Number, b.Number) })
	last := attempts[len(attempts)-1]

	run := Run{
		ID:       last.RunID,
		CaseID:   last.CaseID,
		Name:     last.Name,
		Meta:     last.Meta.Copy(),
		Status:   classify(attempts),
		Start:    attempts[0].Start,
		End:      attempts[0].End,
		Attempts: attempts,
	}
	for _, a := range attempts {
		run.Duration += a.Duration
		if a.Start.Before(run.Start) {
			run.Start = a.Start
		}
		if a.End.After(run.End) {
			run.End = a.End
		}
	}
	run.Elapsed = run.End.Sub(run.Start)

	return run
}

func classify(attempts []Attempt) Status {
	last := attempts[len(attempts)-1].Status
	failed := slices.ContainsFunc(attempts, func(a Attempt) bool { return a.Status == StatusFailed })

	switch {
	case !failed:
		return last
	case last == StatusPassed:
		return StatusFlaky
	default:
		return StatusFailed
	}
}
