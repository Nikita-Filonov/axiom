package axiom

type Skip struct {
	Reason  string
	Enabled bool
}

type SkipOption func(*Skip)

func NewSkip(options ...SkipOption) Skip {
	s := Skip{}
	for _, option := range options {
		option(&s)
	}

	return s
}

func WithSkipEnabled(enabled bool) SkipOption {
	return func(s *Skip) {
		s.Enabled = enabled
	}
}

func WithSkipReason(reason string) SkipOption {
	return func(s *Skip) {
		s.Reason = reason
	}
}

func SkipBecause(reason string) SkipOption {
	return func(s *Skip) {
		s.Enabled = true
		s.Reason = reason
	}
}

func (s *Skip) Copy() Skip {
	return Skip{
		Reason:  s.Reason,
		Enabled: s.Enabled,
	}
}

func (s *Skip) Join(other Skip) Skip {
	result := s.Copy()

	if other.Enabled {
		result.Enabled = true
	}

	if other.Reason != "" {
		result.Reason = other.Reason
	}

	return result
}
