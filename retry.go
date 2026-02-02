package axiom

import (
	"time"
)

type Retry struct {
	Times int
	Delay time.Duration

	TimesSet bool
	DelaySet bool
}

type RetryOption func(*Retry)

func NewRetry(options ...RetryOption) Retry {
	r := Retry{}
	for _, option := range options {
		option(&r)
	}

	return r
}

func WithRetryTimes(times int) RetryOption {
	return func(r *Retry) {
		r.Times = times
		r.TimesSet = true
	}
}

func WithRetryDelay(delay time.Duration) RetryOption {
	return func(r *Retry) {
		r.Delay = delay
		r.DelaySet = true
	}
}

func (r *Retry) Join(other Retry) Retry {
	result := Retry{
		Times:    r.Times,
		Delay:    r.Delay,
		TimesSet: r.TimesSet,
		DelaySet: r.DelaySet,
	}

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
