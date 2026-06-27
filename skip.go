package axiom

type Skip struct {
	Reason     string
	Enabled    bool
	EnabledSet bool
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
		s.EnabledSet = true
	}
}

func WithSkipDisabled() SkipOption {
	return func(s *Skip) {
		s.Enabled = false
		s.EnabledSet = true
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
		s.EnabledSet = true
		s.Reason = reason
	}
}

func (s *Skip) Copy() Skip {
	return Skip{
		Reason:     s.Reason,
		Enabled:    s.Enabled,
		EnabledSet: s.EnabledSet,
	}
}

func (s *Skip) Join(other Skip) Skip {
	result := s.Copy()

	if other.EnabledSet {
		result.Enabled = other.Enabled
		result.EnabledSet = true
	}

	if other.Reason != "" {
		result.Reason = other.Reason
	}

	return result
}
