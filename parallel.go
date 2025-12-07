package axiom

type Parallel struct {
	Enabled bool
}

type ParallelOption func(*Parallel)

func NewParallel(options ...ParallelOption) Parallel {
	p := Parallel{}
	for _, option := range options {
		option(&p)
	}

	return p
}

func WithParallelEnabled() func(*Parallel) {
	return func(p *Parallel) {
		p.Enabled = true
	}
}

func WithParallelDisabled() func(*Parallel) {
	return func(p *Parallel) {
		p.Enabled = false
	}
}

func (p *Parallel) Join(other Parallel) Parallel {
	result := Parallel{Enabled: p.Enabled}

	if other.Enabled {
		result.Enabled = true
	}

	return result
}
