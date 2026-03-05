package axiom

func GetParams[T any](cfg *Config) T {
	v, ok := cfg.Case.Params.(T)
	if !ok {
		cfg.SubT.Fatalf("params: expected type %T, got %T", *new(T), cfg.Case.Params)
	}
	return v
}
