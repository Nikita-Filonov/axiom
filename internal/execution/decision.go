package execution

// DecisionKind identifies how a prepared attempt participates in a run.
type DecisionKind string

const (
	// DecisionRun admits the attempt's hooks and body.
	DecisionRun DecisionKind = "run"

	// DecisionSkip admits reporting wrappers, but not hooks or the body.
	DecisionSkip DecisionKind = "skip"

	// DecisionExclude removes the case from test reporting altogether.
	DecisionExclude DecisionKind = "exclude"
)

// Decision is a resolved execution instruction, not a final test status.
// The caller resolves the decision before submission. A runtime
// testing.T.Skip changes the outcome without changing this decision.
type Decision struct {
	Kind   DecisionKind
	Reason string
}
