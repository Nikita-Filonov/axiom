package axiom

func GetParams[T any](cfg *Config) T {
	v, ok := cfg.Params.(T)
	if !ok {
		cfg.SubT.Fatalf("params: expected type %T, got %T", *new(T), cfg.Params)
	}
	return v
}
