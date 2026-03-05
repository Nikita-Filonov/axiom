package axiom

func GetParams[T any](cfg *Config) T {
	if cfg == nil {
		panic("params: nil config")
	}
	if cfg.Case == nil {
		panic("params: nil case")
	}

	v, ok := cfg.Case.Params.(T)
	if !ok {
		var zero T
		cfg.SubT.Fatalf("params: expected type %T, got %T", zero, cfg.Case.Params)
		return zero
	}
	return v
}
