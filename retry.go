package axiom

import (
	"time"
)

type Retry struct {
	Times int
	Delay time.Duration
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
	}
}

func WithRetryDelay(delay time.Duration) RetryOption {
	return func(r *Retry) {
		r.Delay = delay
	}
}

func (r *Retry) Join(other Retry) Retry {
	result := Retry{
		Times: r.Times,
		Delay: r.Delay,
	}

	if other.Times != 0 {
		result.Times = other.Times
	}
	if other.Delay != 0 {
		result.Delay = other.Delay
	}

	return result
}

func (r *Retry) Normalize() {
	if r.Times <= 0 {
		r.Times = 3
	}
	if r.Delay <= 0 {
		r.Delay = 2 * time.Second
	}
}
