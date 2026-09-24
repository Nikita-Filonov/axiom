package axiom

// Skip describes whether a case should be skipped and why. EnabledSet records
// an explicit choice so a Case can disable a skip inherited from its Runner.
// A Reason alone does not enable skipping.
type Skip struct {
	Reason     string
	Enabled    bool
	EnabledSet bool
}

// SkipOption configures a Skip rule.
type SkipOption func(*Skip)

// NewSkip returns a Skip rule with the supplied options.
func NewSkip(options ...SkipOption) Skip {
	s := Skip{}
	for _, option := range options {
		option(&s)
	}

	return s
}

// WithSkipEnabled explicitly enables or disables skipping.
func WithSkipEnabled(enabled bool) SkipOption {
	return func(s *Skip) {
		s.Enabled = enabled
		s.EnabledSet = true
	}
}

// WithSkipDisabled disables skipping, including an inherited Runner skip.
func WithSkipDisabled() SkipOption {
	return func(s *Skip) {
		s.Enabled = false
		s.EnabledSet = true
	}
}

// WithSkipReason sets a reason without changing whether skipping is enabled.
func WithSkipReason(reason string) SkipOption {
	return func(s *Skip) {
		s.Reason = reason
	}
}

// SkipBecause enables skipping and records its reason.
func SkipBecause(reason string) SkipOption {
	return func(s *Skip) {
		s.Enabled = true
		s.EnabledSet = true
		s.Reason = reason
	}
}

// Copy returns an independent Skip value.
func (s *Skip) Copy() Skip {
	return Skip{
		Reason:     s.Reason,
		Enabled:    s.Enabled,
		EnabledSet: s.EnabledSet,
	}
}

// Join returns a rule where an explicit enabled state and nonempty reason in
// other override the corresponding values in s.
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
