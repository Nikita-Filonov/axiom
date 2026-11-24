package execution

// ResultKind distinguishes test outcomes from selection results.
type ResultKind string

const (
	// AttemptResultCompleted has a non-nil Outcome.
	AttemptResultCompleted ResultKind = "completed"

	// AttemptResultExcluded has no Outcome: a caller filter excluded the case.
	AttemptResultExcluded ResultKind = "excluded"

	// AttemptResultNotSelected has no Outcome: Go did not enter the subtest.
	AttemptResultNotSelected ResultKind = "not-selected"
)

// Result is a terminal result. Only Completed carries a test Outcome.
type Result struct {
	Kind    ResultKind
	Outcome *Outcome
}

func (r Result) clone() Result {
	if r.Outcome != nil {
		outcome := *r.Outcome
		r.Outcome = &outcome
	}

	return r
}
