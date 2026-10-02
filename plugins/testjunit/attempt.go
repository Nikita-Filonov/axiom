package testjunit

import "time"

type status uint8

const (
	statusPassed status = iota
	statusFailed
	statusSkipped
)

type attempt struct {
	SuiteName  string
	TestName   string
	Status     status
	Error      string
	SkipReason string
	Start      time.Time
	Duration   time.Duration
}

type testOutcome interface {
	Failed() bool
	Skipped() bool
}

func outcome(t testOutcome, lifecycleFailed bool) status {
	switch {
	case lifecycleFailed || t.Failed():
		return statusFailed
	case t.Skipped():
		return statusSkipped
	default:
		return statusPassed
	}
}
