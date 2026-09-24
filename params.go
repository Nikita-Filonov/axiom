package axiom

// GetParams returns the current Case's parameters as T. A type mismatch fails
// the active subtest; nil cfg, Case, or SubT causes a panic.
func GetParams[T any](cfg *Config) T {
	if cfg == nil {
		panic("params: nil config")
	}
	if cfg.Case == nil {
		panic("params: nil case")
	}
	if cfg.SubT == nil {
		panic("params: nil subT")
	}

	v, ok := cfg.Case.Params.(T)
	if !ok {
		cfg.SubT.Fatalf("params: expected type %T, got %T", v, cfg.Case.Params)
	}
	return v
}
