package axiom

// Parallel controls whether a case calls testing.T.Parallel. EnabledSet lets
// a Case explicitly disable parallel execution inherited from its Runner.
// Parallel retry attempts to remain sequential within their case.
type Parallel struct {
	Enabled    bool
	EnabledSet bool
}

// ParallelOption configures a Parallel policy.
type ParallelOption func(*Parallel)

// NewParallel returns a Parallel policy with the supplied options.
func NewParallel(options ...ParallelOption) Parallel {
	p := Parallel{}
	for _, option := range options {
		option(&p)
	}

	return p
}

// WithParallelEnabled opts a case into Go's parallel test scheduling.
func WithParallelEnabled() ParallelOption {
	return func(p *Parallel) {
		p.Enabled = true
		p.EnabledSet = true
	}
}

// WithParallelDisabled explicitly opts out of parallel test scheduling.
func WithParallelDisabled() ParallelOption {
	return func(p *Parallel) {
		p.Enabled = false
		p.EnabledSet = true
	}
}

func (p *Parallel) Copy() Parallel {
	return Parallel{Enabled: p.Enabled, EnabledSet: p.EnabledSet}
}

// Join returns a policy where other's explicit choice overrides p.
func (p *Parallel) Join(other Parallel) Parallel {
	result := p.Copy()

	if other.EnabledSet {
		result.Enabled = other.Enabled
		result.EnabledSet = true
	}

	return result
}
