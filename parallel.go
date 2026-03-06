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

func (p *Parallel) Copy() Parallel {
	return Parallel{Enabled: p.Enabled}
}

func (p *Parallel) Join(other Parallel) Parallel {
	result := p.Copy()

	if other.Enabled {
		result.Enabled = true
	}

	return result
}
