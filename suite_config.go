package axiom

type SuiteConfig struct {
	Runner   *Runner
	Parallel bool
}

type SuiteConfigOption func(*SuiteConfig)

func NewSuiteConfig(options ...SuiteConfigOption) SuiteConfig {
	cfg := SuiteConfig{}
	for _, option := range options {
		option(&cfg)
	}

	if cfg.Runner == nil {
		cfg.Runner = NewRunner()
	}

	return cfg
}

func WithSuiteConfigRunner(runner *Runner) SuiteConfigOption {
	return func(cfg *SuiteConfig) { cfg.Runner = runner }
}

func WithSuiteConfigParallel() SuiteConfigOption {
	return func(cfg *SuiteConfig) { cfg.Parallel = true }
}
