package axiom

import (
	"time"
)

// Retry controls the total number of attempts and the delay between them.
// Times defaults to one, and Delay defaults to zero after normalization.
// TimesSet and DelaySet distinguish explicit Case overrides from inherited
// Runner settings. An earlier failure still counts in Go's final test result.
type Retry struct {
	Times int
	Delay time.Duration

	TimesSet bool
	DelaySet bool
}

// RetryOption configures a Retry policy.
type RetryOption func(*Retry)

// NewRetry returns a Retry policy with the supplied options.
func NewRetry(options ...RetryOption) Retry {
	r := Retry{}
	for _, option := range options {
		option(&r)
	}

	return r
}

// WithRetryTimes sets the total number of attempts, including the first run.
// Values below one are normalized to one when the policy is used.
func WithRetryTimes(times int) RetryOption {
	return func(r *Retry) {
		r.Times = times
		r.TimesSet = true
	}
}

// WithRetryDelay sets the wait between attempts. Negative values are
// normalized to zero when the policy is used.
func WithRetryDelay(delay time.Duration) RetryOption {
	return func(r *Retry) {
		r.Delay = delay
		r.DelaySet = true
	}
}

// Copy returns an independent Retry value.
func (r *Retry) Copy() Retry {
	return Retry{
		Times:    r.Times,
		Delay:    r.Delay,
		TimesSet: r.TimesSet,
		DelaySet: r.DelaySet,
	}
}

// Join returns a policy where explicitly set fields in other override r.
func (r *Retry) Join(other Retry) Retry {
	result := r.Copy()

	if other.TimesSet {
		result.Times = other.Times
		result.TimesSet = true
	}
	if other.DelaySet {
		result.Delay = other.Delay
		result.DelaySet = true
	}

	return result
}

// Normalize sets default values and clamps Times to one and Delay to zero.
func (r *Retry) Normalize() {
	if r.TimesSet && r.Times < 1 {
		r.Times = 1
	}
	if !r.TimesSet {
		r.Times = 1
	}
	if r.DelaySet && r.Delay < 0 {
		r.Delay = 0
	}
	if !r.DelaySet {
		r.Delay = 0
	}
}
