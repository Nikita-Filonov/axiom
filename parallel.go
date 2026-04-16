package axiom

type Parallel struct {
	Enabled    bool
	EnabledSet bool
}

type ParallelOption func(*Parallel)

func NewParallel(options ...ParallelOption) Parallel {
	p := Parallel{}
	for _, option := range options {
		option(&p)
	}

	return p
}

func WithParallelEnabled() ParallelOption {
	return func(p *Parallel) {
		p.Enabled = true
		p.EnabledSet = true
	}
}

func WithParallelDisabled() ParallelOption {
	return func(p *Parallel) {
		p.Enabled = false
		p.EnabledSet = true
	}
}

func (p *Parallel) Copy() Parallel {
	return Parallel{Enabled: p.Enabled, EnabledSet: p.EnabledSet}
}

func (p *Parallel) Join(other Parallel) Parallel {
	result := p.Copy()

	if other.EnabledSet {
		result.Enabled = other.Enabled
		result.EnabledSet = true
	}

	return result
}
